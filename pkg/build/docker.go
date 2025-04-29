package build

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	shellquote "github.com/kballard/go-shellquote"
)

func (bb *Build) buildGeneration2(dir string) error {
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

		if err := bb.build(filepath.Join(dir, b.Path), b.Manifest, hash, env); err != nil {
			return err
		}
	}

	for image := range pulls {
		if err := bb.pull(image); err != nil {
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

		for _, to := range tos {
			if err := bb.tag(from, to); err != nil {
				return err
			}

			if bb.EnvWrapper {
				if err := bb.injectConvoxEnv(to); err != nil {
					return err
				}
			}
		}
	}

	pushfroms := []string{}

	for from := range pushes {
		pushfroms = append(pushfroms, from)
	}

	sort.Strings(pushfroms)

	for _, from := range pushfroms {
		to := pushes[from]

		if err := bb.tag(from, to); err != nil {
			return err
		}

		if err := bb.push(to); err != nil {
			return err
		}
	}

	return nil
}

func (bb *Build) build(path, dockerfile string, tag string, env map[string]string) error {
	if path == "" {
		return fmt.Errorf("must have path to build")
	}

	df := filepath.Join(path, dockerfile)

	args := []string{"build"}

	if !bb.Cache {
		args = append(args, "--no-cache")
	}

	args = append(args, "-t", tag)
	args = append(args, "-f", df)
	args = append(args, "--network", "host")

	ba, err := bb.buildArgs(df, env)
	if err != nil {
		return err
	}

	args = append(args, ba...)

	args = append(args, path)

	if err := bb.Exec.Run(bb.writer, "docker", args...); err != nil {
		return err
	}

	data, err := bb.Exec.Execute("docker", "inspect", tag, "--format", "{{json .Config.Entrypoint}}")
	if err != nil {
		return err
	}

	var ep []string

	if err := json.Unmarshal(data, &ep); err != nil {
		return err
	}

	if ep != nil {
		opts := structs.BuildUpdateOptions{
			Entrypoint: options.String(shellquote.Join(ep...)),
		}

		if _, err := bb.Provider.BuildUpdate(bb.App, bb.Id, opts); err != nil {
			return err
		}
	}

	return nil
}

func (bb *Build) buildArgs(dockerfile string, env map[string]string) ([]string, error) {
	fd, err := os.Open(dockerfile)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	s := bufio.NewScanner(fd)

	args := []string{}

	for s.Scan() {
		fields := strings.Fields(strings.TrimSpace(s.Text()))

		if len(fields) < 2 {
			continue
		}

		parts := strings.Split(fields[1], "=")

		switch fields[0] {
		case "FROM":
			if bb.Development && strings.Contains(strings.ToLower(s.Text()), "as development") {
				args = append(args, "--target", "development")
			}
		case "ARG":
			k := strings.TrimSpace(parts[0])
			if v, ok := env[k]; ok {
				args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	return args, nil
}

func (bb *Build) injectConvoxEnv(tag string) error {
	fmt.Fprintf(bb.writer, "Injecting: convox-env\n")

	var cmd []string
	var entrypoint []string

	data, err := bb.Exec.Execute("docker", "inspect", tag, "--format", "{{json .Config.Cmd}}")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &cmd); err != nil {
		return err
	}

	data, err = bb.Exec.Execute("docker", "inspect", tag, "--format", "{{json .Config.Entrypoint}}")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &entrypoint); err != nil {
		return err
	}

	epb, err := json.Marshal(append([]string{"/convox-env"}, entrypoint...))
	if err != nil {
		return err
	}

	epdfs := fmt.Sprintf("FROM %s\nCOPY ./convox-env /convox-env\nENTRYPOINT %s\n", tag, epb)

	if cmd != nil {
		cmdb, err := json.Marshal(cmd)
		if err != nil {
			return err
		}

		epdfs += fmt.Sprintf("CMD %s\n", cmdb)
	}

	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		return err
	}

	if _, err := bb.Exec.Execute("cp", "/go/bin/convox-env", filepath.Join(tmp, "convox-env")); err != nil {
		return err
	}

	epdf := filepath.Join(tmp, "Dockerfile")

	if err := os.WriteFile(epdf, []byte(epdfs), 0644); err != nil {
		return err
	}

	_, err = bb.Exec.Execute("docker", "build", "-t", tag, tmp)
	if err != nil {
		return err
	}

	return nil
}

func (bb *Build) pull(tag string) error {
	fmt.Fprintf(bb.writer, "Running: docker pull %s\n", tag)

	data, err := bb.Exec.Execute("docker", "pull", tag)
	if err != nil {
		return errors.New(strings.TrimSpace(string(data)))
	}

	return nil
}

func (bb *Build) push(tag string) error {
	fmt.Fprintf(bb.writer, "Running: docker push %s\n", tag)

	data, err := bb.Exec.Execute("docker", "push", tag)
	if err != nil {
		return errors.New(strings.TrimSpace(string(data)))
	}

	return nil
}

func (bb *Build) tag(from, to string) error {
	fmt.Fprintf(bb.writer, "Running: docker tag %s %s\n", from, to)

	data, err := bb.Exec.Execute("docker", "tag", from, to)
	if err != nil {
		return errors.New(strings.TrimSpace(string(data)))
	}

	return nil
}
