package source

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type SourceZip struct {
	URL string
}

func (s *SourceZip) Fetch(out io.Writer) (string, error) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	r, err := urlReader(s.URL)
	if err != nil {
		return "", err
	}

	defer r.Close()

	atmp, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	defer os.RemoveAll(atmp)

	a := filepath.Join(atmp, "archive.zip")

	fd, err := os.OpenFile(a, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}

	n, err := io.Copy(fd, r)
	if err != nil {
		return "", err
	}

	fd, err = os.Open(a)
	if err != nil {
		return "", err
	}

	zr, err := zip.NewReader(fd, n)
	if err != nil {
		return "", err
	}

	for _, file := range zr.File {
		fmt.Printf("file = %+v\n", file)

		path := filepath.Join(tmp, file.Name)

		if file.FileInfo().IsDir() {
			if err := os.Mkdir(path, os.FileMode(file.FileInfo().Mode())); err != nil {
				return "", err
			}
		} else {
			fd, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.FileMode(file.FileInfo().Mode()))
			if err != nil {
				return "", err
			}

			defer fd.Close()

			fr, err := file.Open()
			if err != nil {
				return "", err
			}

			defer fr.Close()

			if _, err := io.Copy(fd, fr); err != nil {
				return "", err
			}

			fr.Close()
			fd.Close()
		}
	}

	fmt.Printf("tmp = %+v\n", tmp)

	return tmp, nil
}
