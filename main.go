package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/convox/build/Godeps/_workspace/src/github.com/convox/cli/manifest"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "build: turn a convox application into an ami\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s <name> <source>\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n  build example-sinatra https://github.com/convox-examples/sinatra.git\n")
	}
}

func main() {
	id := flag.String("id", "", "tag the build with this id")
	push := flag.String("push", "", "push build to this prefix when done")
	auth := flag.String("auth", "", "auth token for push")

	flag.Parse()

	l := len(flag.Args())

	if l < 2 {
		flag.Usage()
		os.Exit(0)
	}

	args := flag.Args()

	app := positional(args, 0)
	source := positional(args, 1)

	dir, err := clone(source, app)

	if err != nil {
		die(err)
	}

	m, err := manifest.Generate(dir)

	if err != nil {
		die(err)
	}

	data, err := m.Raw()

	if err != nil {
		die(err)
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		fmt.Printf("manifest|%s\n", scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		die(err)
	}

	manifest.Stdout = prefixWriter("build")
	manifest.Stderr = manifest.Stdout

	if err != nil {
		die(err)
	}

	errors := m.Build(app, dir)

	if len(errors) > 0 {
		die(errors[0])
	}

	if *push != "" {
		manifest.Stdout = prefixWriter("push")
		manifest.Stderr = manifest.Stdout

		errors := m.Push(app, *push, *auth, *id)

		if len(errors) > 0 {
			die(errors[0])
		}
	}
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
}

func clone(source, app string) (string, error) {
	tmp, err := ioutil.TempDir("", "repo")

	if err != nil {
		return "", err
	}

	clone := filepath.Join(tmp, "clone")

	switch {
	case isDir(source):
		return source, nil
	case source == "-":
		err := extractTarball(os.Stdin, clone)

		if err != nil {
			return "", err
		}
	default:
		if err = writeFile("/usr/local/bin/git-restore-mtime", "git-restore-mtime", 0755, nil); err != nil {
			return "", err
		}

		err = run("git", tmp, "git", "clone", source, clone)

		if err != nil {
			return "", err
		}

		err = run("git", clone, "/usr/local/bin/git-restore-mtime", ".")

		if err != nil {
			return "", err
		}
	}

	return clone, nil
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

			fd, err := os.OpenFile(join, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))

			if err != nil {
				return err
			}

			defer fd.Close()

			_, err = io.Copy(fd, tr)

			if err != nil {
				return err
			}

			err = os.Chtimes(join, time.Now(), header.ModTime)

			if err != nil {
				return err
			}
		default:
			fmt.Printf("unknown Typeflag: %d %d\n", header.Typeflag, tar.TypeReg)
		}
	}
}

func prefixWriter(prefix string) io.Writer {
	r, w := io.Pipe()
	go prefixReader(r, prefix)
	return w
}

func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func scanLinesWithMax(data []byte, atEof bool) (advance int, token []byte, err error) {
	if atEof && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, dropCR(data[0:i]), nil
	}

	if len(data) > 2048 {
		return 2048, dropCR(data[0:2048]), nil
	}

	if atEof {
		return len(data), dropCR(data), nil
	}

	return 0, nil, nil
}

func prefixReader(r io.Reader, prefix string) {
	scanner := bufio.NewScanner(r)

	scanner.Split(scanLinesWithMax)

	for scanner.Scan() {
		fmt.Printf("%s|%s\n", prefix, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("error|%s\n", err.Error())
	}
}

func run(prefix, dir string, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir

	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	if err != nil {
		return err
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

func isDir(dir string) bool {
	fd, err := os.Open(dir)

	if err != nil {
		return false
	}

	stat, err := fd.Stat()

	if err != nil {
		return false
	}

	return stat.IsDir()
}

func positional(args []string, n int) string {
	if len(args) > n {
		return args[n]
	} else {
		return ""
	}
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
