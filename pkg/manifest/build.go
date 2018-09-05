package manifest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/builder/dockerignore"
)

type BuildSource struct {
	Local  string
	Remote string
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

	path, err := filepath.Abs(filepath.Join(root, s.Build.Path, s.Build.Manifest))
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("no such file: %s", filepath.Join(s.Build.Path, s.Build.Manifest))
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

	svc, err := m.Service(service)
	if err != nil {
		return nil, err
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
					local := filepath.Join(svc.Build.Path, parts[1])
					remote := replaceEnv(parts[2], env)

					// if remote == "." || strings.HasSuffix(remote, "/") {
					//   remote = filepath.Join(remote, filepath.Base(local))
					// }

					if wd != "" && !filepath.IsAbs(remote) {
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

				data, err = exec.Command("docker", "inspect", parts[1], "--format", "{{.Config.WorkingDir}}").CombinedOutput()
				if err != nil {
					return nil, err
				}

				wd = strings.TrimSpace(string(data))
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

		stat, err := os.Stat(abs)
		if err != nil {
			return nil, err
		}

		if stat.IsDir() && !strings.HasSuffix(abs, "/") {
			abs = abs + "/"
		}

		bs[i].Local = abs

		if bs[i].Remote == "." {
			bs[i].Remote = wd
		}
	}

	bss := []BuildSource{}

	for i := range bs {
		contained := false

		for j := i + 1; j < len(bs); j++ {
			if strings.HasPrefix(bs[i].Local, bs[j].Local) {
				if bs[i].Remote == bs[j].Remote {
					contained = true
					break
				}

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

	return bss, nil
}

func replaceEnv(s string, env map[string]string) string {
	for k, v := range env {
		s = strings.Replace(s, fmt.Sprintf("${%s}", k), v, -1)
		s = strings.Replace(s, fmt.Sprintf("$%s", k), v, -1)
	}

	return s
}
