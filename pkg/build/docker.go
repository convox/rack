package build

// NOTE: classic‑runtime builder helpers (Docker‑in‑Docker).  This file is only
// compiled when BUILD_RUNTIME != "daemonless".

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	shellquote "github.com/kballard/go-shellquote"
)

// -----------------------------------------------------------------------------
// top‑level Generation‑2 dispatcher (Docker runtime)
// -----------------------------------------------------------------------------

func (bb *Build) buildGeneration2(dir string) error {
	configPath := filepath.Join(dir, bb.Manifest)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest %s not found: %w", bb.Manifest, err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	env, err := helpers.AppEnvironment(bb.Provider, bb.App)
	if err != nil {
		return fmt.Errorf("load app env: %w", err)
	}
	// merge CLI‑provided build args → env map (overrides runtime env)
	for _, v := range bb.BuildArgs {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid build‑arg %q (expected k=v)", v)
		}
		env[parts[0]] = parts[1]
	}

	m, err := manifest.Load(data, env)
	if err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	prefix := fmt.Sprintf("%s/%s", bb.Rack, bb.App)

	// work sets
	builds := map[string]manifest.ServiceBuild{}
	pulls := map[string]struct{}{}
	pushes := map[string]string{}
	tags := map[string][]string{}

	for _, s := range m.Services {
		hash := s.BuildHash(bb.Id)
		target := fmt.Sprintf("%s:%s.%s", prefix, s.Name, bb.Id)

		if s.Image != "" {
			pulls[s.Image] = struct{}{}
			tags[s.Image] = append(tags[s.Image], target)
		} else {
			builds[hash] = s.Build
			tags[hash] = append(tags[hash], target)
		}

		if bb.Push != "" {
			pushes[target] = fmt.Sprintf("%s:%s.%s", bb.Push, s.Name, bb.Id)
		}
	}

	// 1) docker build for each unique hash
	for hash, b := range builds {
		if err := bb.build(filepath.Join(dir, b.Path), b.Manifest, hash, env); err != nil {
			return err // already wrapped upstream
		}
	}

	// 2) docker pull for external images
	for image := range pulls {
		if err := bb.pull(image); err != nil {
			return err
		}
	}

	// 3) docker tag (+ optional env injection)
	tagSrcs := keys(tags)
	sort.Strings(tagSrcs)

	for _, src := range tagSrcs {
		for _, dst := range tags[src] {
			if err := bb.tag(src, dst); err != nil {
				return err
			}
			if bb.EnvWrapper {
				if err := bb.injectConvoxEnv(dst); err != nil {
					return err
				}
			}
		}
	}

	// 4) docker push if requested
	pushSrcs := keys(pushes)
	sort.Strings(pushSrcs)
	for _, src := range pushSrcs {
		dst := pushes[src]
		if err := bb.tag(src, dst); err != nil {
			return err
		}
		if err := bb.push(dst); err != nil {
			return err
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// docker build helper
// -----------------------------------------------------------------------------

func (bb *Build) build(path, dockerfile, tag string, env map[string]string) error {
	if path == "" {
		return fmt.Errorf("build path cannot be empty")
	}

	df := filepath.Join(path, dockerfile)

	args := []string{"build"}
	if !bb.Cache {
		args = append(args, "--no-cache")
	}

	args = append(args, "-t", tag, "-f", df, "--network", "host")

	ba, err := bb.buildArgs(df, env)
	if err != nil {
		return err // already wrapped inside buildArgs
	}
	args = append(args, ba...)
	args = append(args, path)

	if err := bb.Exec.Run(bb.writer, "docker", args...); err != nil {
		return fmt.Errorf("docker build %s: %w", path, err)
	}

	// extract entrypoint for later release info
	data, err := bb.Exec.Execute("docker", "inspect", tag, "--format", "{{json .Config.Entrypoint}}")
	if err != nil {
		return fmt.Errorf("docker inspect entrypoint: %w", err)
	}

	var ep []string
	if err := json.Unmarshal(data, &ep); err != nil {
		return fmt.Errorf("parse entrypoint json: %w", err)
	}

	if len(ep) > 0 {
		if _, err := bb.Provider.BuildUpdate(bb.App, bb.Id, structs.BuildUpdateOptions{
			Entrypoint: options.String(shellquote.Join(ep...)),
		}); err != nil {
			return fmt.Errorf("save entrypoint: %w", err)
		}
	}

	return nil
}

// -----------------------------------------------------------------------------
// docker build‑arg helper
// -----------------------------------------------------------------------------

var (
	reArg  = regexp.MustCompile(`(?i)^\s*ARG\s+([^=\s]+)`)
	reFrom = regexp.MustCompile(`(?i)^\s*FROM\s+.*\bAS\s+development\b`)
)

// buildArgs returns CLI "--build-arg" flags derived from ARG statements that
// have matches in the supplied env map.  It also adds "--target development"
// automatically when the Build is in development mode and the Dockerfile has a
// stage named "development".
func (bb *Build) buildArgs(dockerfile string, env map[string]string) ([]string, error) {
	fd, err := os.Open(dockerfile)
	if err != nil {
		return nil, fmt.Errorf("open Dockerfile: %w", err)
	}
	defer fd.Close()

	scanner := bufio.NewScanner(fd)
	var args []string

	for scanner.Scan() {
		line := scanner.Text()
		if bb.Development && reFrom.MatchString(line) {
			args = append(args, "--target", "development")
		}

		if m := reArg.FindStringSubmatch(line); m != nil {
			key := strings.TrimSpace(m[1])
			if val, ok := env[key]; ok {
				args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, val))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan Dockerfile: %w", err)
	}

	return args, nil
}

// -----------------------------------------------------------------------------
// misc helpers
// -----------------------------------------------------------------------------

func (bb *Build) injectConvoxEnv(tag string) error {
	bb.Printf("Injecting: convox-env\n")

	var (
		cmd        []string
		entrypoint []string
	)

	data, err := bb.Exec.Execute("docker", "inspect", tag, "--format", "{{json .Config.Cmd}}")
	if err != nil {
		return fmt.Errorf("inspect cmd: %w", err)
	}
	if err := json.Unmarshal(data, &cmd); err != nil {
		return fmt.Errorf("parse cmd json: %w", err)
	}

	data, err = bb.Exec.Execute("docker", "inspect", tag, "--format", "{{json .Config.Entrypoint}}")
	if err != nil {
		return fmt.Errorf("inspect entrypoint: %w", err)
	}
	if err := json.Unmarshal(data, &entrypoint); err != nil {
		return fmt.Errorf("parse entrypoint json: %w", err)
	}

	epJSON, err := json.Marshal(append([]string{"/convox-env"}, entrypoint...))
	if err != nil {
		return fmt.Errorf("marshal entrypoint: %w", err)
	}

	dockerfile := fmt.Sprintf("FROM %s\nCOPY ./convox-env /convox-env\nENTRYPOINT %s\n", tag, epJSON)
	if cmd != nil {
		cmdJSON, _ := json.Marshal(cmd) // cannot error
		dockerfile += fmt.Sprintf("CMD %s\n", cmdJSON)
	}

	tmp, err := os.MkdirTemp("", "convox-env-")
	if err != nil {
		return fmt.Errorf("tempdir: %w", err)
	}
	defer os.RemoveAll(tmp)

	if _, err := bb.Exec.Execute("cp", "/go/bin/convox-env", filepath.Join(tmp, "convox-env")); err != nil {
		return fmt.Errorf("copy convox-env: %w", err)
	}

	if err := os.WriteFile(filepath.Join(tmp, "Dockerfile"), []byte(dockerfile), 0644); err != nil {
		return fmt.Errorf("write Dockerfile: %w", err)
	}

	if _, err := bb.Exec.Execute("docker", "build", "-t", tag, tmp); err != nil {
		return fmt.Errorf("rebuild with env wrapper: %w", err)
	}

	return nil
}

func (bb *Build) pull(tag string) error {
	bb.Printf("Running: docker pull %s\n", tag)
	if data, err := bb.Exec.Execute("docker", "pull", tag); err != nil {
		return errors.New(strings.TrimSpace(string(data)))
	}
	return nil
}

func (bb *Build) push(tag string) error {
	bb.Printf("Running: docker push %s\n", tag)
	if data, err := bb.Exec.Execute("docker", "push", tag); err != nil {
		return errors.New(strings.TrimSpace(string(data)))
	}
	return nil
}

func (bb *Build) tag(from, to string) error {
	bb.Printf("Running: docker tag %s %s\n", from, to)
	if data, err := bb.Exec.Execute("docker", "tag", from, to); err != nil {
		return errors.New(strings.TrimSpace(string(data)))
	}
	return nil
}

// keys returns map keys as slice.
func keys[M ~map[K]V, K comparable, V any](m M) []K {
	ks := make([]K, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
