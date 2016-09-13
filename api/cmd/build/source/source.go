package source

import (
	"io"
	"net/http"
	"net/url"
	"os"
)

type Source interface {
	Fetch() (string, error)
}

func urlReader(url_ string) (io.ReadCloser, error) {
	u, err := url.Parse(url_)
	if err != nil {
		return nil, err
	}

	if u.Scheme == "file" {
		fd, err := os.Open(u.Path)
		if err != nil {
			return nil, err
		}

		return fd, nil
	}

	req, err := http.Get(url_)
	if err != nil {
		return nil, err
	}

	return req.Body, nil
}
