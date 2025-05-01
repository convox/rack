package build

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/kballard/go-shellquote"
)

// workspaceDir is the directory baked into the builder image that Kaniko writes to.
const workspaceDir = "/workspace"
const convoxEnvBinary = "/busybox/convox-env"

// tarPathFor generates a safe on‑disk tar filename for an image tag by replacing
// characters that are illegal in file names.
func tarPathFor(tag string) string {
	safe := strings.NewReplacer("/", "_", ":", "_").Replace(tag)
	return filepath.Join(workspaceDir, safe+".tar")
}

func injectedTarPathFor(tag string) string {
	safe := strings.NewReplacer("/", "_", ":", "_").Replace(tag)
	return filepath.Join(workspaceDir, "injected-"+safe+".tar")
}

// ──────────────────────────────────────────────────────────────────────────────
// Top‑level build entrypoint for generation‑2 / daemonless.
// ──────────────────────────────────────────────────────────────────────────────

func (bb *Build) buildGeneration2Daemonless(dir string) error {
	config := filepath.Join(dir, bb.Manifest)
	if _, err := os.Stat(config); os.IsNotExist(err) {
		return fmt.Errorf("no such file: %s", bb.Manifest)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	env, err := helpers.AppEnvironment(bb.Provider, bb.App)
	if err != nil {
		return fmt.Errorf("loading app env: %w", err)
	}

	// merge CLI‑supplied build args into env map so buildArgs() picks them up
	for _, v := range bb.BuildArgs {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid build arg: %s", v)
		}
		env[parts[0]] = parts[1]
	}

	m, err := manifest.Load(data, env)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	prefix := fmt.Sprintf("%s/%s", bb.Rack, bb.App)

	builds := map[string]manifest.ServiceBuild{}
	pulls := map[string]bool{}
	pushes := map[string]string{}
	tags := map[string][]string{}

	for _, s := range m.Services {
		hash := s.BuildHash(bb.Id)
		to := fmt.Sprintf("%s:%s.%s", prefix, s.Name, bb.Id)

		if s.Image != "" {
			pulls[s.Image] = true
			tags[s.Image] = append(tags[s.Image], to)
		} else {
			builds[hash] = s.Build
			tags[hash] = append(tags[hash], to)
		}

		if bb.Push != "" {
			pushes[to] = fmt.Sprintf("%s:%s.%s", bb.Push, s.Name, bb.Id)
		}
	}

	// ─── Build phases ────────────────────────────────────────────────────────

	for hash, b := range builds {
		if err := bb.buildDaemonless(filepath.Join(dir, b.Path), b.Manifest, hash, env); err != nil {
			return err
		}
	}

	for image := range pulls {
		if err := bb.pullDaemonless(image); err != nil {
			return err
		}
	}

	tagFroms := make([]string, 0, len(tags))
	for from := range tags {
		tagFroms = append(tagFroms, from)
	}
	sort.Strings(tagFroms)

	for _, from := range tagFroms {
		tos := tags[from]

		if bb.EnvWrapper {
			if err := bb.injectConvoxEnvDaemonless(from); err != nil {
				return err
			}
		}

		for _, to := range tos {
			destination := pushes[to]
			if err := bb.tagAndPushDaemonless(from, destination); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildDaemonless builds a single Dockerfile context with Kaniko and stores the
// resulting OCI‑image tarball in workspaceDir.
func (bb *Build) buildDaemonless(path, dockerfile, tag string, env map[string]string) error {
	contextDir := "dir://" + path
	tarPath := tarPathFor(tag)

	args := []string{
		"--dockerfile", dockerfile,
		"--context", contextDir,
		"--tarPath", tarPath,
		"--destination", tag,
		"--no-push",
		"--cache=false",
	}

	df := filepath.Join(path, dockerfile)
	ba, err := bb.buildArgs(df, env)
	if err != nil {
		return err
	}
	args = append(args, ba...)

	if err := bb.Exec.Run(bb.writer, "/kaniko/executor", args...); err != nil {
		return err
	}

	// Extract entrypoint for release metadata
	img, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return fmt.Errorf("loading kaniko image tar: %w", err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return fmt.Errorf("reading image config: %w", err)
	}

	if ep := cfg.Config.Entrypoint; ep != nil {
		_, err = bb.Provider.BuildUpdate(bb.App, bb.Id, structs.BuildUpdateOptions{
			Entrypoint: options.String(shellquote.Join(ep...)),
		})
		if err != nil {
			return fmt.Errorf("saving entrypoint: %w", err)
		}
	}

	return nil
}

// injectConvoxEnvDaemonless mutates the already‑built tarball by prepending the
// /convox-env wrapper and rewriting the entrypoint.
func (bb *Build) injectConvoxEnvDaemonless(tag string) error {
	bb.Printf("Injecting: convox-env\n")

	originalTar := tarPathFor(tag)
	injectedTar := injectedTarPathFor(tag)

	img, err := tarball.ImageFromPath(originalTar, nil)
	if err != nil {
		return fmt.Errorf("load image: %w", err)
	}

	bin, err := os.ReadFile(convoxEnvBinary)
	if err != nil {
		return fmt.Errorf("read %s: %w", convoxEnvBinary, err)
	}

	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(singleFileTar("/convox-env", bin))), nil
	})
	if err != nil {
		return fmt.Errorf("create layer: %w", err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	// adjust entrypoint first, then apply in single mutate.Config call later
	cfg.Config.Entrypoint = append([]string{"/convox-env"}, cfg.Config.Entrypoint...)

	// apply config update
	img, err = mutate.Config(img, cfg.Config)
	if err != nil {
		return fmt.Errorf("mutate config: %w", err)
	}

	// append new layer
	img, err = mutate.Append(img, mutate.Addendum{Layer: layer})
	if err != nil {
		return fmt.Errorf("append layer: %w", err)
	}

	ref, err := name.NewTag(tag)
	if err != nil {
		return fmt.Errorf("parse tag: %w", err)
	}

	if err := tarball.WriteToFile(injectedTar, ref, img); err != nil {
		return fmt.Errorf("write injected tar: %w", err)
	}

	return nil
}

// pullDaemonless pulls a remote image to the local cache so that later we can
// re‑tag and push it. Uses registry auth when needed.
func (bb *Build) pullDaemonless(tag string) error {
	bb.Printf("Running: docker pull %s\n", tag)

	auth, err := ecrAuthenticator()
	if err != nil {
		return err
	}
	if _, err := crane.Pull(tag, crane.WithAuth(auth)); err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	return nil
}

func (bb *Build) tagAndPushDaemonless(from, to string) error {
	bb.Printf("Running: tag and push %s\n", to)

	imgTar := tarPathFor(from)
	img, err := tarball.ImageFromPath(imgTar, nil)
	if err != nil {
		return fmt.Errorf("load tar: %w", err)
	}

	auth, err := ecrAuthenticator()
	if err != nil {
		return err
	}
	if err := crane.Push(img, to, crane.WithAuth(auth)); err != nil {
		return fmt.Errorf("push: %w", err)
	}

	bb.Printf("Pushed %s\n", to)
	return nil
}

// ---------------------------------------------------------------------------
// ECR authentication helpers
// ---------------------------------------------------------------------------

var (
	ecrOnce sync.Once
	ecrAuth authn.Authenticator
	ecrErr  error
)

func ecrAuthenticator() (authn.Authenticator, error) {
	ecrOnce.Do(func() {
		sess, err := session.NewSession()
		if err != nil {
			ecrErr = fmt.Errorf("aws session: %w", err)
			return
		}

		svc := ecr.New(sess)
		out, err := svc.GetAuthorizationToken(nil)
		if err != nil {
			ecrErr = fmt.Errorf("ecr auth token: %w", err)
			return
		}
		if len(out.AuthorizationData) == 0 {
			ecrErr = fmt.Errorf("empty authorization data")
			return
		}

		token, err := base64.StdEncoding.DecodeString(*out.AuthorizationData[0].AuthorizationToken)
		if err != nil {
			ecrErr = fmt.Errorf("decode token: %w", err)
			return
		}
		parts := strings.SplitN(string(token), ":", 2)
		if len(parts) != 2 {
			ecrErr = fmt.Errorf("invalid token format")
			return
		}

		ecrAuth = &authn.Basic{Username: parts[0], Password: parts[1]}
	})

	return ecrAuth, ecrErr
}

// singleFileTar returns a tar stream containing a single file at path.
func singleFileTar(path string, data []byte) []byte {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	hdr := &tar.Header{
		Name:     strings.TrimPrefix(path, "/"),
		Mode:     0755,
		Size:     int64(len(data)),
		Typeflag: tar.TypeReg,
	}
	_ = tw.WriteHeader(hdr)
	_, _ = tw.Write(data)

	return buf.Bytes()
}
