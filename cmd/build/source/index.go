package source

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/convox/rack/structs"
	"github.com/convox/rack/provider"
)

type SourceIndex struct {
	URL string
}

func (s *SourceIndex) Fetch(out io.Writer) (string, error) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}

	r, err := urlReader(s.URL)
	if err != nil {
		return "", err
	}

	defer r.Close()

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return "", err
	}

	var index structs.Index

	if err := json.Unmarshal(data, &index); err != nil {
		return "", err
	}

	if err := provider.FromEnv().IndexDownload(&index, tmp); err != nil {
		return "", err
	}

	return tmp, nil
}
