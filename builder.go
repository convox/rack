package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ()

type Builder struct {
	GitHubToken string
}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) Build(repo, name, ref, push, id string) error {
	clone, err := b.compose(repo, name, ref)

	if err != nil {
		return err
	}

	if push != "" {
		b.push(clone, push, name, id)
	}

	return nil
}

func (b *Builder) clone(repo, name, ref string) (string, error) {
	tmp, err := ioutil.TempDir("", "repo")

	if err != nil {
		return "", err
	}

	clone := filepath.Join(tmp, "clone")

	if err = writeFile(os.Getenv("HOME"), ".netrc", 0600, map[string]string{"{{GITHUB_TOKEN}}": b.GitHubToken}); err != nil {
		return "", err
	}

	if err = writeFile("/usr/local/bin", "git-restore-mtime", 0755, nil); err != nil {
		return "", err
	}

	err = b.run("git", tmp, "git", "clone", repo, clone)

	if err != nil {
		return "", err
	}

	if ref != "" {
		err = b.run("git", clone, "git", "checkout", ref)

		if err != nil {
			return "", err
		}
	}

	err = b.run("git", clone, "/usr/local/bin/git-restore-mtime", ".")

	if err != nil {
		return "", err
	}

	return clone, nil
}

func (b *Builder) compose(repo, name, ref string) (string, error) {
	dir, err := b.clone(repo, name, ref)

	if err != nil {
		return "", err
	}

	manifest, err := ioutil.ReadFile(filepath.Join(dir, "docker-compose.yml"))

	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(bytes.NewReader(manifest))

	for scanner.Scan() {
		fmt.Printf("manifest|%s\n", scanner.Text())
	}

	b.run("compose", dir, "docker-compose", "-p", "app", "build")
	b.run("compose", dir, "docker-compose", "-p", "app", "pull")

	return dir, nil
}

func (b *Builder) run(prefix, dir string, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	if err != nil {
		return err
	}

	fmt.Printf("%s|RUNNING: %s %s\n", prefix, command, strings.Join(args, " "))

	cmd.Start()

	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		fmt.Printf("%s|%s\n", prefix, scanner.Text())
	}

	err = cmd.Wait()

	if err != nil {
		fmt.Printf("%s|error: %s\n", prefix, err)
	}

	return err
}

func (b *Builder) push(dir, target, name, id string) error {
	manifest, err := ReadManifest(dir)

	if err != nil {
		return err
	}

	for ps, entry := range *manifest {
		from := fmt.Sprintf("app_%s", ps)

		if entry.Image != "" {
			from = entry.Image
		}

		to := fmt.Sprintf("%s/%s-%s", target, name, ps)

		if id != "" {
			to = fmt.Sprintf("%s:%s", to, id)
		}

		b.run("push", dir, "docker", "tag", "-f", from, to)
		b.run("push", dir, "docker", "push", to)
	}

	return nil
}

func dataRaw(path string) ([]byte, error) {
	return Asset(fmt.Sprintf("data/%s", path))
}

func writeFile(dir, name string, perms os.FileMode, replacements map[string]string) error {
	data, err := Asset(fmt.Sprintf("data/%s", name))

	if err != nil {
		return err
	}

	sdata := string(data)

	if replacements != nil {
		for key, val := range replacements {
			sdata = strings.Replace(sdata, key, val, -1)
		}
	}

	return ioutil.WriteFile(filepath.Join(dir, name), []byte(sdata), perms)
}
