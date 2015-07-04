package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
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

func (b *Builder) Build(repo, name, ref, push, auth, id string) error {
	prefix := generateId("a", 8)

	clone, err := b.compose(prefix, repo, name, ref)

	if err != nil {
		return err
	}

	if push != "" {
		err := b.push(prefix, clone, push, name, auth, id)

		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Builder) clone(repo, name, ref string) (string, error) {
	tmp, err := ioutil.TempDir("", "repo")

	if err != nil {
		return "", err
	}

	clone := filepath.Join(tmp, "clone")

	if repo == "-" {
		extractTarball(os.Stdin, clone)
	} else {
		if err = writeFile(filepath.Join(os.Getenv("HOME"), ".netrc"), "netrc", 0600, map[string]string{"{{GITHUB_TOKEN}}": b.GitHubToken}); err != nil {
			return "", err
		}

		if err != nil {
			return "", err
		}

		if err = writeFile("/usr/local/bin/git-restore-mtime", "git-restore-mtime", 0755, nil); err != nil {
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
	}

	return clone, nil
}

func (b *Builder) compose(prefix, repo, name, ref string) (string, error) {
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

	b.run("compose", dir, "docker-compose", "-p", prefix, "build")
	b.run("compose", dir, "docker-compose", "-p", prefix, "pull")

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

	if prefix != "auth" {
		fmt.Printf("%s|RUNNING: %s %s\n", prefix, command, strings.Join(args, " "))
	}

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

func (b *Builder) push(prefix, dir, target, name, auth, id string) error {
	manifest, err := ReadManifest(dir)

	if err != nil {
		return err
	}

	if auth != "" {
		err := b.run("auth", dir, "docker", "login", "-e", "user@convox.io", "-u", "convox", "-p", auth, target)

		if err != nil {
			return err
		}
	}

	for ps, entry := range *manifest {
		from := fmt.Sprintf("%s_%s", prefix, ps)

		if entry.Image != "" {
			from = entry.Image
		}

		to := fmt.Sprintf("%s/%s-%s", target, name, ps)

		if id != "" {
			to = fmt.Sprintf("%s:%s", to, id)
		}

		err := b.run("push", dir, "docker", "tag", "-f", from, to)

		if err != nil {
			return err
		}

		err = b.run("push", dir, "docker", "push", to)

		if err != nil {
			return err
		}
	}

	return nil
}

func dataRaw(path string) ([]byte, error) {
	return Asset(fmt.Sprintf("data/%s", path))
}

func extractTarball(r io.Reader, base string) error {
	gz, err := gzip.NewReader(r)

	if err != nil {
		return err
	}

	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()

		if err != nil {
			if err == io.EOF {
				return nil
			} else {
				return err
			}
		}

		rel := header.Name
		join := filepath.Join(base, rel)

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(join, 0755)
		case tar.TypeReg, tar.TypeRegA:
			dir := filepath.Dir(join)

			os.MkdirAll(dir, 0755)

			fd, err := os.OpenFile(join, os.O_CREATE, os.FileMode(header.Mode))

			if err != nil {
				return err
			}

			defer fd.Close()

			_, err = io.Copy(fd, tr)

			if err != nil {
				return err
			}
		default:
			fmt.Printf("unknown Typeflag: %d %d\n", header.Typeflag, tar.TypeReg)
		}
	}
}

var idAlphabet = []rune("abcdefghijklmnopqrstuvwxyz")

func generateId(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = idAlphabet[rand.Intn(len(idAlphabet))]
	}
	return prefix + string(b)
}

func writeFile(target, name string, perms os.FileMode, replacements map[string]string) error {
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

	return ioutil.WriteFile(target, []byte(sdata), perms)
}
