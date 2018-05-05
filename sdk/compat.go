package sdk

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/convox/rack/structs"
	"github.com/convox/stdsdk"
)

func (c *Client) BuildImportMultipart(app string, r io.Reader) (*structs.Build, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	ro := stdsdk.RequestOptions{
		Files: stdsdk.Files{
			"image": data,
		},
	}

	var b *structs.Build

	if err := c.Post(fmt.Sprintf("/apps/%s/builds", app), ro, &b); err != nil {
		return nil, err
	}

	return b, nil
}

func (c *Client) BuildImportUrl(app string, r io.Reader) (*structs.Build, error) {
	o, err := c.ObjectStore(app, "", r, structs.ObjectStoreOptions{})
	if err != nil {
		return nil, err
	}

	ro := stdsdk.RequestOptions{
		Params: stdsdk.Params{
			"url": o.Url,
		},
	}

	var b *structs.Build

	if err := c.Post(fmt.Sprintf("/apps/%s/builds/import", app), ro, &b); err != nil {
		return nil, err
	}

	return b, nil
}

func (c *Client) ProcessRunAttached(app, service string, rw io.ReadWriter, opts structs.ProcessRunOptions) (int, error) {
	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return 0, err
	}

	ro.Body = rw

	code, err := c.WebsocketExit(fmt.Sprintf("/apps/%s/processes/%s/run", app, service), ro, rw)
	if err != nil {
		return 0, err
	}

	return code, nil
}
