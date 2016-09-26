package client

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type Index map[string]IndexItem

type IndexItem struct {
	Name    string      `json:"name"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"mtime"`
	Size    int         `json:"-"`
}

func (c *Client) IndexMissing(index Index) ([]string, error) {
	var missing []string

	data, err := json.Marshal(index)

	if err != nil {
		return nil, err
	}

	params := Params{
		"index": string(data),
	}

	err = c.Post(fmt.Sprintf("/index/diff"), params, &missing)

	if err != nil {
		return nil, err
	}

	return missing, nil
}

type IndexUpdateOptions struct {
	Progress Progress
}

// IndexUpdate uploads a tarball of changes to the index
func (c *Client) IndexUpdate(update io.Reader, opts IndexUpdateOptions) error {
	popts := PostMultipartOptions{
		Files: map[string]io.Reader{
			"update": update,
		},
		Progress: opts.Progress,
	}

	return c.PostMultipart("/index/update", popts, nil)
}
