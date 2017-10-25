package manifest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/convox/rack/types"
	"github.com/docker/docker/builder/dockerignore"
)

type BuildOptions struct {
	Cache       string
	Development bool
	Env         Environment
	Push        string
	Root        string
	Stdout      io.Writer
	Stderr      io.Writer
}

type BuildSource struct {
	Local  string
	Remote string
}

func (m *Manifest) Build(root, prefix string, tag string, opts BuildOptions) error {
	builds := map[string]ServiceBuild{}
	pulls := map[string]bool{}
	pushes := map[string]string{}
	tags := map[string][]string{}

	for _, s := range m.Services {
		hash := s.BuildHash()
		to := fmt.Sprintf("%s/%s:%s", prefix, s.Name, tag)

		if s.Image != "" {
			pulls[s.Image] = true
			tags[s.Image] = append(tags[s.Image], to)
		} else {
			builds[hash] = s.Build
			tags[hash] = append(tags[hash], to)
		}

		if opts.Push != "" {
			pushes[to] = fmt.Sprintf("%s:%s.%s", opts.Push, s.Name, tag)
		}
	}

	for hash, b := range builds {
		if opts.Cache != "" {
			lcd := filepath.Join(opts.Root, b.Path, ".cache", "build")
			rcd := filepath.Join(opts.Cache, hash)

			exec.Command("mkdir", "-p", lcd).Run()

			rls, err := ioutil.ReadDir(rcd)
			if err != nil {
				pe, ok := err.(*os.PathError)
				if !ok {
					return err
				}

				if pe.Err.Error() != "no such file or directory" {
					return err
				}
			}

			if len(rls) > 0 {
				exec.Command("rm", "-rf", lcd).Run()
			}

			exec.Command("cp", "-a", rcd, lcd).Run()
		}

		if err := build(b, hash, opts); err != nil {
			return err
		}

		if opts.Cache != "" {
			exec.Command("rm", "-rf", filepath.Join(opts.Cache, "*")).Run()

			name, err := types.Key(32)
			if err != nil {
				return err
			}

			opts.dockerq("create", "--name", name, hash)
			opts.dockerq("cp", fmt.Sprintf("%s:/var/cache/build", name), filepath.Join(opts.Cache, hash))
			opts.dockerq("rm", name)
		}
	}

	for image := range pulls {
		if err := pull(image, opts); err != nil {
			return err
		}
	}

	for from, tos := range tags {
		for _, to := range tos {
			if err := opts.docker("tag", from, to); err != nil {
				return err
			}
		}
	}

	for from, to := range pushes {
		if err := opts.docker("tag", from, to); err != nil {
			return err
		}

		fmt.Fprintf(opts.Stdout, "pushing: %s\n", to)

		if err := opts.dockerq("push", to); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manifest) BuildIgnores(root, service string) ([]string, error) {
	ignore := []string{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		ip := filepath.Join(path, ".dockerignore")

		if _, err := os.Stat(ip); os.IsNotExist(err) {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		fd, err := os.Open(ip)
		if err != nil {
			return err
		}

		lines, err := dockerignore.ReadAll(fd)
		if err != nil {
			return err
		}

		for _, line := range lines {
			ignore = append(ignore, filepath.Join(rel, line))
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return ignore, nil
}

func (m *Manifest) BuildDockerfile(root, service string) ([]byte, error) {
	s, err := m.Service(service)
	if err != nil {
		return nil, err
	}

	if s.Image != "" {
		return nil, nil
	}

	path, err := filepath.Abs(filepath.Join(root, s.Build.Path, "Dockerfile"))
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("no such file: %s", filepath.Join(s.Build.Path, "Dockerfile"))
	}

	return ioutil.ReadFile(path)
}

func (m *Manifest) BuildSources(root, service string) ([]BuildSource, error) {
	data, err := m.BuildDockerfile(root, service)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return []BuildSource{}, nil
	}

	bs := []BuildSource{}
	env := map[string]string{}
	wd := ""

	s := bufio.NewScanner(bytes.NewReader(data))

	for s.Scan() {
		parts := strings.Fields(s.Text())

		if len(parts) < 1 {
			continue
		}

		switch strings.ToUpper(parts[0]) {
		case "ADD", "COPY":
			if len(parts) > 2 {
				u, err := url.Parse(parts[1])
				if err != nil {
					return nil, err
				}

				switch u.Scheme {
				case "http", "https":
					// do nothing
				default:
					local := parts[1]
					remote := replaceEnv(parts[2], env)

					if remote == "." || strings.HasSuffix(remote, "/") {
						remote = filepath.Join(remote, filepath.Base(local))
					}

					if wd != "" {
						remote = filepath.Join(wd, remote)
					}

					bs = append(bs, BuildSource{Local: local, Remote: remote})
				}
			}
		case "ENV":
			if len(parts) > 2 {
				env[parts[1]] = parts[2]
			}
		case "FROM":
			if len(parts) > 1 {
				var ee []string

				data, err := exec.Command("docker", "inspect", parts[1], "--format", "{{json .Config.Env}}").CombinedOutput()
				if err != nil {
					return nil, err
				}

				if err := json.Unmarshal(data, &ee); err != nil {
					return nil, err
				}

				for _, e := range ee {
					parts := strings.SplitN(e, "=", 2)

					if len(parts) == 2 {
						env[parts[0]] = parts[1]
					}
				}
			}
		case "WORKDIR":
			if len(parts) > 1 {
				wd = replaceEnv(parts[1], env)
			}
		}
	}

	for i := range bs {
		abs, err := filepath.Abs(bs[i].Local)
		if err != nil {
			return nil, err
		}

		bs[i].Local = abs
	}

	bss := []BuildSource{}

	for i := range bs {
		contained := false

		for j := i + 1; j < len(bs); j++ {
			if strings.HasPrefix(bs[i].Local, bs[j].Local) {
				rl, err := filepath.Rel(bs[j].Local, bs[i].Local)
				if err != nil {
					return nil, err
				}

				rr, err := filepath.Rel(bs[j].Remote, bs[i].Remote)
				if err != nil {
					return nil, err
				}

				if rl == rr {
					contained = true
					break
				}
			}
		}

		if !contained {
			bss = append(bss, bs[i])
		}
	}

	// return nil, fmt.Errorf("stop")

	return bss, nil
}

func replaceEnv(s string, env map[string]string) string {
	for k, v := range env {
		s = strings.Replace(s, fmt.Sprintf("${%s}", k), v, -1)
		s = strings.Replace(s, fmt.Sprintf("$%s", k), v, -1)
	}

	return s
}

func build(b ServiceBuild, tag string, opts BuildOptions) error {
	if b.Path == "" {
		return fmt.Errorf("must have path to build")
	}

	args := []string{"build"}

	args = append(args, "-t", tag)

	path, err := filepath.Abs(filepath.Join(opts.Root, b.Path))
	if err != nil {
		return err
	}

	df := filepath.Join(path, "Dockerfile")

	if opts.Development {
		data, err := ioutil.ReadFile(df)
		if err != nil {
			return err
		}

		dev := bytes.SplitN(data, []byte("## convox:production"), 2)

		if err := ioutil.WriteFile(df, dev[0], 0644); err != nil {
			return err
		}
	}

	ba, err := buildArgs(df, opts)
	if err != nil {
		return err
	}

	args = append(args, ba...)

	args = append(args, path)

	message(opts.Stdout, "building: %s", b.Path)

	return opts.docker(args...)
}

func buildArgs(dockerfile string, opts BuildOptions) ([]string, error) {
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
			if opts.Development && strings.Contains(strings.ToLower(s.Text()), "as development") {
				args = append(args, "--target", "development")
			}
		case "ARG":
			k := strings.TrimSpace(parts[0])
			if v, ok := opts.Env[k]; ok {
				args = append(args, "--build-arg", fmt.Sprintf("%s=%s", k, v))
			}
		}
	}

	return args, nil
}

func pull(image string, opts BuildOptions) error {
	message(opts.Stdout, "pulling: %s", image)

	if err := opts.docker("pull", image); err != nil {
		return err
	}

	return nil
}

func (o BuildOptions) docker(args ...string) error {
	message(o.Stdout, "running: docker %s", strings.Join(args, " "))

	cmd := exec.Command("docker", args...)

	cmd.Stdout = o.Stdout
	cmd.Stderr = o.Stderr

	return cmd.Run()
}

func (o BuildOptions) dockerq(args ...string) error {
	return exec.Command("docker", args...).Run()
}
