package build

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

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
	kanikoTarPath = "/workspace/%s.tar"
	convoxEnvPath = "/usr/local/bin/convox-env"
)

func (bb *Build) buildDaemonless(path, dockerfile string, tag string, env map[string]string) error {
	// make sure this is the directory you’ve mounted into the Kaniko container
	// Kaniko expects a “dir://” prefix for local contexts:
	contextDir := "dir://" + path
	bb.Printf("Using Kaniko to build %s in %s", tag, contextDir)

	useCache := "--cache=true"
	if !bb.Cache {
		useCache = "--cache=false"
	}
	tarPath := fmt.Sprintf(kanikoTarPath, tag)
	buildArgs := []string{
		"--dockerfile", dockerfile,
		"--context", contextDir,
		"--no-push",
		"--tarPath", tarPath,
		useCache,
	}

	df := filepath.Join(path, dockerfile)
	ba, err := bb.buildArgs(df, env)
	if err != nil {
		return err
	}
	buildArgs = append(buildArgs, ba...)

	err = bb.Exec.Run(bb.writer, "/kaniko/executor", buildArgs...)
	if err != nil {
		return err
	}

	bb.Printf("Running Kaniko build...")

	// Load image from Kaniko output to extract entrypoint
	img, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return fmt.Errorf("loading image from Kaniko tar: %w", err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

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
	bb.Printf("Injecting convox-env and pushing final image...")
	fmt.Fprintf(bb.writer, "Injecting: convox-env\n")

	// Load image from Kaniko output
	tarPath := fmt.Sprintf(kanikoTarPath, tag)
	img, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return fmt.Errorf("loading image from Kaniko tar: %w", err)
	}

	// Read /convox-env binary
	bin, err := os.ReadFile(convoxEnvPath)
	if err != nil {
		return fmt.Errorf("reading convox-env binary: %w", err)
	}

	// Create a new tar layer with convox-env
	layer, err := tarball.LayerFromOpener(func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(singleFileTar("/convox-env", bin))), nil
	})
	if err != nil {
		return fmt.Errorf("creating layer: %w", err)
	}

	// Update entrypoint
	cfg, err := img.ConfigFile()
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	cfg.Config.Entrypoint = append([]string{"/convox-env"}, cfg.Config.Entrypoint...)

	img, err = mutate.Append(img, mutate.Addendum{Layer: layer})
	if err != nil {
		return fmt.Errorf("appending layer: %w", err)
	}

	img, err = mutate.Config(img, cfg.Config)
	if err != nil {
		return fmt.Errorf("mutating config: %w", err)
	}

	// Save the modified image to a tarball
	ref, err := name.NewTag(tag)
	if err != nil {
		return fmt.Errorf("parsing tag %q: %w", tag, err)
	}
	if err := tarball.WriteToFile(tarPath, ref, img, nil); err != nil {
		return fmt.Errorf("writing tarball: %w", err)
	}

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

func (b *Build) pullDeamonless(tag string) error {
	b.Printf("Pulling: %s\n", tag)

	_, err := crane.Pull(tag, crane.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}
	return nil
}

func (b *Build) pushDeamonless(tag string) error {
	b.Printf("Pushing: %s\n", tag)
	// Load image from Kaniko output
	tarPath := fmt.Sprintf(kanikoTarPath, tag)
	img, err := tarball.ImageFromPath(tarPath, nil)
	if err != nil {
		return fmt.Errorf("loading image from Kaniko tar: %w", err)
	}

	return crane.Push(img, tag, crane.WithAuthFromKeychain(authn.DefaultKeychain))
}

func (b *Build) tagAndPushDaemonless(from, to string) error {
	b.Printf("Tagging: %s → %s\n", from, to)

	img, err := crane.Pull(from, crane.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return fmt.Errorf("pull for tag failed: %w", err)
	}

	return crane.Push(img, to, crane.WithAuthFromKeychain(authn.DefaultKeychain))
}
