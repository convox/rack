package source

import (
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/convox/rack/provider"
)

type Source interface {
	Fetch(out io.Writer) (string, error)
}

func urlReader(url_ string) (io.ReadCloser, error) {
	u, err := url.Parse(url_)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "file":
		fd, err := os.Open(u.Path)
		if err != nil {
			return nil, err
		}
		return fd, nil
	case "object":
		return provider.FromEnv().ObjectFetch(u.Host, u.Path)
	}

	req, err := http.Get(url_)
	if err != nil {
		return nil, err
	}

	return req.Body, nil
}
