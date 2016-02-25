package client

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Index map[string]IndexItem

type IndexItem struct {
	Name    string      `json:"name"`
	Mode    os.FileMode `json:"mode"`
	ModTime time.Time   `json:"mtime"`
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

func (c *Client) IndexUpload(hash string, data []byte) error {
	files := map[string][]byte{
		"data": data,
	}

	return c.PostMultipart(fmt.Sprintf("/index/file/%s", hash), files, nil, nil)
}
