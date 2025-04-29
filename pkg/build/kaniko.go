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

const (
	kanikoTarPath   = "/workspace/%s.tar"
	injectedTarPath = "/workspace/injected-%s.tar"
	convoxEnvPath   = "/busybox/convox-env"
)

func (bb *Build) buildGeneration2Daemonless(dir string) error {
	config := filepath.Join(dir, bb.Manifest)

	if _, err := os.Stat(config); os.IsNotExist(err) {
		return fmt.Errorf("no such file: %s", bb.Manifest)
	}

	data, err := os.ReadFile(config)
	if err != nil {
		return err
	}

	env, err := helpers.AppEnvironment(bb.Provider, bb.App)
	if err != nil {
		return err
	}

	for _, v := range bb.BuildArgs {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid build args: %s", v)
		}
		env[parts[0]] = parts[1]
	}

	m, err := manifest.Load(data, env)
	if err != nil {
		return err
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

	for hash, b := range builds {
		bb.Printf("Building: %s\n", b.Path)

		if err := bb.buildDaemonless(filepath.Join(dir, b.Path), b.Manifest, hash, env); err != nil {
			return err
		}
	}

	for image := range pulls {
		if err := bb.pullDaemonless(image); err != nil {
			return err
		}
	}

	tagfroms := []string{}
	for from := range tags {
		tagfroms = append(tagfroms, from)
	}

	sort.Strings(tagfroms)
	for _, from := range tagfroms {
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

func (bb *Build) buildDaemonless(path, dockerfile string, tag string, env map[string]string) error {
	fmt.Printf("Path: %s\n", path)
	fmt.Printf("Dockerfile: %s\n", dockerfile)
	fmt.Printf("Tag: %s\n", tag)
	fmt.Printf("Env: %v\n", env)

	// make sure this is the directory you’ve mounted into the Kaniko container
	// Kaniko expects a “dir://” prefix for local contexts:
	contextDir := "dir://" + path
	bb.Printf("Using Kaniko to build %s in %s", tag, contextDir)

	useCache := "--cache=false"
	tarPath := fmt.Sprintf(kanikoTarPath, tag)
	fmt.Printf("building to %s\n", tarPath)
	buildArgs := []string{
		"--dockerfile", dockerfile,
		"--context", contextDir,
		"--tarPath", tarPath,
		"--destination", tag,
		useCache,
		"--no-push",
	}

	df := filepath.Join(path, dockerfile)
	ba, err := bb.buildArgs(df, env)
	if err != nil {
		return err
	}
	buildArgs = append(buildArgs, ba...)

	fmt.Printf("calling kaniko with args: %s\n", buildArgs)
	err = bb.Exec.Run(bb.writer, "/kaniko/executor", buildArgs...)
	if err != nil {
		return err
	}

	bb.Printf("Running Kaniko build...")

	// Load image from Kaniko output to extract entrypoint
	img, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return fmt.Errorf("buildDaemonless: loading image from Kaniko %s tar: %w", tarPath, err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	fmt.Printf("Entrypoint: %v\n", cfg.Config.Entrypoint)
	if cfg.Config.Entrypoint != nil {
		opts := structs.BuildUpdateOptions{
			Entrypoint: options.String(shellquote.Join(cfg.Config.Entrypoint...)),
		}

		if _, err := bb.Provider.BuildUpdate(bb.App, bb.Id, opts); err != nil {
			return fmt.Errorf("saving entrypoint: %w", err)
		}
	}

	return nil
}

func (bb *Build) injectConvoxEnvDaemonless(tag string) error {
	bb.Printf("Injecting convox-env into image built by Kaniko...")

	// Prepare paths
	originalTarPath := fmt.Sprintf(kanikoTarPath, tag)
	toTarPath := fmt.Sprintf(injectedTarPath, tag)

	bb.Printf("Original image tarball: %s", originalTarPath)
	bb.Printf("Injected image tarball: %s", toTarPath)

	// Load the original image built by Kaniko
	img, err := tarball.ImageFromPath(originalTarPath, nil)
	if err != nil {
		return fmt.Errorf("loading original Kaniko image tarball: %w", err)
	}

	// Read convox-env binary from disk
	bin, err := os.ReadFile(convoxEnvPath)
	if err != nil {
		return fmt.Errorf("reading convox-env binary: %w", err)
	}

	// Create a new layer that adds convox-env
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(singleFileTar("/convox-env", bin))), nil
	})
	if err != nil {
		return fmt.Errorf("creating new layer for convox-env: %w", err)
	}

	// Get current image config
	cfg, err := img.ConfigFile()
	if err != nil {
		return fmt.Errorf("reading image config: %w", err)
	}

	// Modify entrypoint to add convox-env
	cfg.Config.Entrypoint = append([]string{"/convox-env"}, cfg.Config.Entrypoint...)

	// Apply mutations: add new layer and update config
	img, err = mutate.Append(img, mutate.Addendum{Layer: layer})
	if err != nil {
		return fmt.Errorf("appending convox-env layer: %w", err)
	}

	img, err = mutate.Config(img, cfg.Config)
	if err != nil {
		return fmt.Errorf("mutating image config: %w", err)
	}

	// Prepare the new reference for the injected tarball
	ref, err := name.NewTag(tag)
	if err != nil {
		return fmt.Errorf("parsing tag: %w", err)
	}

	// Write the modified image to a NEW tarball
	bb.Printf("Writing mutated image tarball to: %s", toTarPath)

	if err := tarball.WriteToFile(toTarPath, ref, img); err != nil {
		return fmt.Errorf("writing injected tarball: %w", err)
	}

	bb.Printf("Successfully injected convox-env and saved to: %s", toTarPath)

	return nil
}

func singleFileTar(path string, data []byte) []byte {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	hdr := &tar.Header{
		Name:     path[1:], // strip leading slash
		Mode:     0755,
		Size:     int64(len(data)),
		Typeflag: tar.TypeReg,
	}
	tw.WriteHeader(hdr)
	tw.Write(data)

	return buf.Bytes()
}

func (b *Build) pullDaemonless(tag string) error {
	b.Printf("Pulling: %s\n", tag)

	// Fetch ECR auth token
	authenticator, err := ecrAuthenticator()
	if err != nil {
		return fmt.Errorf("tagAndPushDaemonless: failed to authenticate to ECR: %w", err)
	}

	_, err = crane.Pull(tag, crane.WithAuth(authenticator))
	if err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}

	return nil
}

func (b *Build) tagAndPushDaemonless(from, to string) error {
	b.Printf("Tagging: %s → %s\n", from, to)

	// Load image
	tarPath := fmt.Sprintf(kanikoTarPath, from)
	img, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return fmt.Errorf("tagAndPushDaemonless: loading image from Kaniko tar: %w", err)
	}

	// Fetch ECR auth token
	authenticator, err := ecrAuthenticator()
	if err != nil {
		return fmt.Errorf("tagAndPushDaemonless: failed to authenticate to ECR: %w", err)
	}

	// Push the image with ECR auth
	if err := crane.Push(img, to, crane.WithAuth(authenticator)); err != nil {
		return fmt.Errorf("tagAndPushDaemonless: failed to push image: %w", err)
	}

	b.Printf("Successfully pushed image to %s\n", to)
	return nil
}

func ecrAuthenticator() (authn.Authenticator, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, fmt.Errorf("creating aws session: %w", err)
	}

	svc := ecr.New(sess)
	authTokenOutput, err := svc.GetAuthorizationToken(nil)
	if err != nil {
		return nil, fmt.Errorf("getting ECR auth token: %w", err)
	}

	if len(authTokenOutput.AuthorizationData) == 0 {
		return nil, fmt.Errorf("no authorization data returned")
	}

	authData := authTokenOutput.AuthorizationData[0]
	token, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		return nil, fmt.Errorf("decoding authorization token: %w", err)
	}

	parts := strings.SplitN(string(token), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid authorization token format")
	}

	return &authn.Basic{
		Username: parts[0],
		Password: parts[1],
	}, nil
}
