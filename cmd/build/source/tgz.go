package source

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type SourceTgz struct {
	URL string
}

func (s *SourceTgz) Fetch(out io.Writer) (string, error) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	r, err := urlReader(s.URL)
	if err != nil {
		return "", err
	}

	defer r.Close()

	gz, err := gzip.NewReader(r)
	if err != nil {
		return "", err
	}

	tr := tar.NewReader(gz)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		path := filepath.Join(tmp, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeReg:
			fd, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return "", err
			}

			defer fd.Close()

			if _, err := io.Copy(fd, tr); err != nil {
				return "", err
			}

			fd.Close()
		case tar.TypeDir:
			if _, stat := os.Stat(path); !os.IsNotExist(stat) {
				continue
			}

			if err := os.Mkdir(path, os.FileMode(hdr.Mode)); err != nil {
				return "", err
			}
		}
	}

	return tmp, nil
}
