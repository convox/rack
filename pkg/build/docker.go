package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// loginForDocker performs Docker registry login using docker CLI
func loginForDocker(bb *Build, auth map[string]struct {
	Username string
	Password string
}) error {
	for host, entry := range auth {
		buf := &strings.Builder{}

		err := bb.Exec.Stream(buf, strings.NewReader(entry.Password), "docker", "login", "-u", entry.Username, "--password-stdin", host)

		bb.Printf("Authenticating %s: %s\n", host, strings.TrimSpace(buf.String()))

		if err != nil {
			return err
		}
	}

	return nil
}

// buildWithDocker builds a Docker image using the Docker CLI
func (bb *Build) buildWithDocker(path, dockerfile string, tag string, env map[string]string) error {
	args := []string{"build"}

	if !bb.Cache {
		args = append(args, "--no-cache")
	}

	args = append(args, "-t", tag)
	args = append(args, "-f", dockerfile)
	args = append(args, "--network", "host")

	ba, err := bb.buildArgs(dockerfile, env)
	if err != nil {
		return err
	}

	args = append(args, ba...)

	args = append(args, path)

	if err := bb.Exec.Run(bb.writer, "docker", args...); err != nil {
		return err
	}

	return nil
}

// pull retrieves an image from a Docker registry
func (bb *Build) pull(image string) error {
	fmt.Fprintf(bb.writer, "Running: docker pull %s\n", image)

	data, err := bb.Exec.Execute("docker", "pull", image)
	if err != nil {
		return errors.New(strings.TrimSpace(string(data)))
	}

	return nil
}

// push sends an image to a Docker registry
func (bb *Build) push(tag string) error {
	fmt.Fprintf(bb.writer, "Running: docker push %s\n", tag)

	data, err := bb.Exec.Execute("docker", "push", tag)
	if err != nil {
		return errors.New(strings.TrimSpace(string(data)))
	}

	return nil
}

// tag adds a tag to an existing Docker image
func (bb *Build) tag(from, to string) error {
	fmt.Fprintf(bb.writer, "Running: docker tag %s %s\n", from, to)

	data, err := bb.Exec.Execute("docker", "tag", from, to)
	if err != nil {
		return errors.New(strings.TrimSpace(string(data)))
	}

	return nil
}

// injectConvoxEnv adds the convox-env executable to a Docker image
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
