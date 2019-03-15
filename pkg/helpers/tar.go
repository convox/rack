package helpers

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/pkg/archive"
)

func Archive(file string) (io.Reader, error) {
	opts := &archive.TarOptions{
		IncludeFiles: []string{file},
	}

	r, err := archive.TarWithOptions("/", opts)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func RebaseArchive(r io.Reader, src, dst string) (io.Reader, error) {
	tr := tar.NewReader(r)

	var buf bytes.Buffer

	tw := tar.NewWriter(&buf)

	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if !strings.HasPrefix(h.Name, "/") {
			h.Name = fmt.Sprintf("/%s", h.Name)
		}

		if !strings.HasPrefix(h.Name, src) {
			continue
		}

		h.Name = filepath.Join(dst, strings.TrimPrefix(h.Name, src))

		tw.WriteHeader(h)

		if _, err := io.Copy(tw, tr); err != nil {
			return nil, err
		}
	}

	tw.Close()

	return &buf, nil
}

func Tarball(dir string) ([]byte, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	sym, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(filepath.Join(sym, ".dockerignore"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	excludes, err := dockerignore.ReadAll(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	defer os.Chdir(cwd)

	if err := os.Chdir(sym); err != nil {
		return nil, err
	}

	opts := &archive.TarOptions{
		Compression:     archive.Gzip,
		ExcludePatterns: excludes,
		IncludeFiles:    []string{"."},
	}

	r, err := archive.TarWithOptions(sym, opts)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(r)
}

func Unarchive(r io.Reader, target string) error {
	tr := tar.NewReader(r)

	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		file := filepath.Join(target, h.Name)

		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(file, os.FileMode(h.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
				return err
			}

			fd, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY, os.FileMode(h.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(fd, tr); err != nil {
				return err
			}
		}
	}
}
