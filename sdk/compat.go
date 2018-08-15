package sdk

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"

	"github.com/convox/rack/structs"
	"github.com/convox/stdsdk"
)

func (c *Client) AppParametersGet(name string) (map[string]string, error) {
	var params map[string]string

	if err := c.Get(fmt.Sprintf("/apps/%s/parameters", name), stdsdk.RequestOptions{}, &params); err != nil {
		return nil, err
	}

	return params, nil
}

func (c *Client) AppParametersSet(name string, params map[string]string) error {
	ro := stdsdk.RequestOptions{
		Params: stdsdk.Params{},
	}

	for k, v := range params {
		ro.Params[k] = v
	}

	if err := c.Post(fmt.Sprintf("/apps/%s/parameters", name), ro, nil); err != nil {
		return err
	}

	return nil
}

func (c *Client) BuildCreateUpload(app string, r io.Reader, opts structs.BuildCreateOptions) (*structs.Build, error) {
	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	ro.Params["cache"] = fmt.Sprintf("%t", (opts.NoCache == nil || !*opts.NoCache))

	if ro.Params["manifest"] != nil {
		ro.Params["config"] = ro.Params["manifest"]
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	ro.Files = stdsdk.Files{"source": data}

	var b *structs.Build

	if err := c.Post(fmt.Sprintf("/apps/%s/builds", app), ro, &b); err != nil {
		return nil, err
	}

	return b, nil
}

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

func (c *Client) CertificateCreateClassic(pub string, key string, opts structs.CertificateCreateOptions) (*structs.Certificate, error) {
	var err error

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	ro.Params["public"] = pub
	ro.Params["private"] = key

	var v *structs.Certificate

	err = c.Post(fmt.Sprintf("/certificates"), ro, &v)

	return v, err
}

func (c *Client) EnvironmentSet(app string, env []byte) (*structs.Release, error) {
	req, err := c.Request("POST", fmt.Sprintf("/apps/%s/environment", app), stdsdk.RequestOptions{Body: bytes.NewReader(env)})
	if err != nil {
		return nil, err
	}

	res, err := c.HandleRequest(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	id := res.Header.Get("Release-Id")

	r, err := c.ReleaseGet(app, id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) EnvironmentUnset(app string, key string) (*structs.Release, error) {
	req, err := c.Request("DELETE", fmt.Sprintf("/apps/%s/environment/%s", app, key), stdsdk.RequestOptions{})
	if err != nil {
		return nil, err
	}

	res, err := c.HandleRequest(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	id := res.Header.Get("Release-Id")

	r, err := c.ReleaseGet(app, id)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) FormationGet(app string) (structs.Services, error) {
	var fs []struct {
		Balancer string
		Count    int
		Cpu      int
		Memory   int
		Name     string
		Ports    []int
	}

	if err := c.Get(fmt.Sprintf("/apps/%s/formation", app), stdsdk.RequestOptions{}, &fs); err != nil {
		return nil, err
	}

	ss := structs.Services{}

	for _, f := range fs {
		var ssls []struct {
			Certificate string
			Process     string
			Port        int
		}

		if err := c.Get(fmt.Sprintf("/apps/%s/ssl", app), stdsdk.RequestOptions{}, &ssls); err != nil {
			return nil, err
		}

		s := structs.Service{
			Count:  f.Count,
			Cpu:    f.Cpu,
			Domain: f.Balancer,
			Memory: f.Memory,
			Name:   f.Name,
			Ports:  []structs.ServicePort{},
		}

		for _, p := range f.Ports {
			cert := ""

			for _, ssl := range ssls {
				if ssl.Process == s.Name && ssl.Port == p {
					cert = ssl.Certificate
					break
				}
			}

			s.Ports = append(s.Ports, structs.ServicePort{Balancer: p, Certificate: cert})
		}

		ss = append(ss, s)
	}

	return ss, nil
}

func (c *Client) FormationUpdate(app string, service string, opts structs.ServiceUpdateOptions) error {
	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return err
	}

	if err := c.Post(fmt.Sprintf("/apps/%s/formation/%s", app, service), ro, nil); err != nil {
		return err
	}

	return nil
}

func (c *Client) InstanceShellClassic(id string, rw io.ReadWriter, opts structs.InstanceShellOptions) (int, error) {
	var err error

	// bug in old rack switched these
	opts.Height, opts.Width = opts.Width, opts.Height

	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return 0, err
	}

	ro.Headers["Terminal"] = "xterm"

	ro.Body = rw

	var v int

	v, err = c.WebsocketExit(fmt.Sprintf("/instances/%s/ssh", id), ro, rw)
	if err != nil {
		return 0, err
	}

	return v, err
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

func (c *Client) ProcessRunDetached(app, service string, opts structs.ProcessRunOptions) (string, error) {
	ro, err := stdsdk.MarshalOptions(opts)
	if err != nil {
		return "", err
	}

	var ret struct {
		Pid string
	}

	if err := c.Post(fmt.Sprintf("/apps/%s/processes/%s/run", app, service), ro, &ret); err != nil {
		return "", err
	}

	return ret.Pid, nil
}

func (c *Client) ResourceCreateClassic(kind string, opts structs.ResourceCreateOptions) (*structs.Resource, error) {
	ro := stdsdk.RequestOptions{
		Params: stdsdk.Params{
			"type": kind,
		},
	}

	if opts.Name != nil {
		ro.Params["name"] = *opts.Name
	} else {
		ro.Params["name"] = fmt.Sprintf("%s-%d", kind, (rand.Intn(8999) + 1000))
	}

	if opts.Parameters != nil {
		for k, v := range opts.Parameters {
			ro.Params[k] = v
		}
	}

	var r *structs.Resource

	if err := c.Post("/resources", ro, &r); err != nil {
		return nil, err
	}

	return r, nil
}

func (c *Client) ResourceUpdateClassic(name string, opts structs.ResourceUpdateOptions) (*structs.Resource, error) {
	ro := stdsdk.RequestOptions{
		Params: stdsdk.Params{},
	}

	if opts.Parameters != nil {
		for k, v := range opts.Parameters {
			ro.Params[k] = v
		}
	}

	var r *structs.Resource

	if err := c.Put(fmt.Sprintf("/resources/%s", name), ro, &r); err != nil {
		return nil, err
	}

	return r, nil
}
