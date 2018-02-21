package sdk

import (
	"fmt"
	"io"
	"strings"
)

func (c *Client) FilesDelete(app, pid string, files []string) error {
	ro := RequestOptions{
		Params: Params{
			"files": strings.Join(files, ","),
		},
	}

	return c.Delete(fmt.Sprintf("/apps/%s/processes/%s/files", app, pid), ro, nil)
}

func (c *Client) FilesUpload(app, pid string, r io.Reader) error {
	ro := RequestOptions{
		Body: r,
	}

	return c.Post(fmt.Sprintf("/apps/%s/processes/%s/files", app, pid), ro, nil)
}
