package router

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	sortableTime = "20060102.150405.000000000"
)

type Client struct {
	Host string
}

func NewClient(host string) *Client {
	return &Client{
		Host: host,
	}
}

func (c *Client) EndpointCreate(rack, hostname, protocol string, port int) error {
	ro := RequestOptions{
		Params: Params{
			"port":     strconv.Itoa(port),
			"protocol": protocol,
		},
	}

	return c.post(fmt.Sprintf("/racks/%s/hosts/%s/endpoints", rack, hostname), ro, nil)
}

func (c *Client) EndpointGet(rack, hostname string, port int) (*Endpoint, error) {
	var e Endpoint

	if err := c.get(fmt.Sprintf("/racks/%s/hosts/%s/endpoints/%d", rack, hostname, port), RequestOptions{}, &e); err != nil {
		return nil, err
	}

	return &e, nil
}

func (c *Client) HostCreate(rack, hostname string) error {
	ro := RequestOptions{
		Params: Params{
			"hostname": hostname,
		},
	}

	return c.post(fmt.Sprintf("/racks/%s/hosts", rack), ro, nil)
}

func (c *Client) HostGet(rack, hostname string) (*Host, error) {
	var h Host

	if err := c.get(fmt.Sprintf("/racks/%s/hosts/%s", rack, hostname), RequestOptions{}, &h); err != nil {
		return nil, err
	}

	return &h, nil
}

func (c *Client) RackCreate(name, endpoint string) error {
	ro := RequestOptions{
		Params: Params{
			"endpoint": endpoint,
			"name":     name,
		},
	}

	return c.post("/racks", ro, nil)
}

func (c *Client) RackGet(name string) (*Rack, error) {
	var r Rack

	if err := c.get(fmt.Sprintf("/racks/%s", name), RequestOptions{}, &r); err != nil {
		return nil, err
	}

	return &r, nil
}

func (c *Client) TargetAdd(rack, hostname string, port int, target string) error {
	ro := RequestOptions{
		Params: Params{
			"target": target,
		},
	}

	return c.post(fmt.Sprintf("/racks/%s/hosts/%s/endpoints/%d/targets/add", rack, hostname, port), ro, nil)
}

func (c *Client) TargetList(rack, hostname string, port int) ([]string, error) {
	var t []string

	if err := c.get(fmt.Sprintf("/racks/%s/hosts/%s/endpoints/%d/targets", rack, hostname, port), RequestOptions{}, &t); err != nil {
		return nil, err
	}

	return t, nil
}

func (c *Client) TargetRemove(rack, hostname string, port int, target string) error {
	ro := RequestOptions{
		Params: Params{
			"target": target,
		},
	}

	return c.post(fmt.Sprintf("/racks/%s/hosts/%s/endpoints/%d/targets/delete", rack, hostname, port), ro, nil)
}

func (c *Client) Terminate() error {
	return c.post("/terminate", RequestOptions{}, nil)
}

func (c *Client) Version() (string, error) {
	var v struct {
		Version string
	}

	if err := c.get("/version", RequestOptions{}, &v); err != nil {
		return "", err
	}

	return v.Version, nil
}

func (c *Client) get(path string, opts RequestOptions, out interface{}) error {
	req, err := c.request("GET", path, opts)
	if err != nil {
		return err
	}

	res, err := c.handleRequest(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	return unmarshalReader(res.Body, out)
}

func (c *Client) post(path string, opts RequestOptions, out interface{}) error {
	req, err := c.request("POST", path, opts)
	if err != nil {
		return err
	}

	res, err := c.handleRequest(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	return unmarshalReader(res.Body, out)
}

func (c *Client) client() *http.Client {
	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &http.Client{
		Transport: t,
	}
}

func (c *Client) request(method, path string, opts RequestOptions) (*http.Request, error) {
	qs, err := opts.Querystring()
	if err != nil {
		return nil, err
	}

	r, err := opts.Reader()
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("https://%s%s%s?%s", c.Host, path, qs)

	req, err := http.NewRequest(method, endpoint, r)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "*/*")
	req.Header.Set("Connection", "close")
	req.Header.Set("Content-Type", opts.ContentType())

	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

func (c *Client) handleRequest(req *http.Request) (*http.Response, error) {
	res, err := c.client().Do(req)
	if err != nil {
		return nil, err
	}

	if err := responseError(res); err != nil {
		return nil, err
	}

	return res, nil
}

func marshalValues(vv map[string]interface{}) (url.Values, error) {
	u := url.Values{}

	for k, v := range vv {
		switch t := v.(type) {
		case string:
			u.Set(k, t)
		case []string:
			for _, s := range t {
				u.Add(k, s)
			}
		case time.Time:
			u.Set(k, t.Format(sortableTime))
		default:
			return nil, fmt.Errorf("unknown param type: %T", t)
		}
	}

	return u, nil
}

func responseError(res *http.Response) error {
	if res.StatusCode < 400 {
		return nil
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var e struct {
		Error string
	}

	if err := json.Unmarshal(data, &e); err == nil && e.Error != "" {
		return errors.New(e.Error)
	}

	msg := strings.TrimSpace(string(data))

	if len(msg) > 0 {
		return fmt.Errorf(msg)
	}

	return fmt.Errorf("response status %d", res.StatusCode)
}

func unmarshalReader(r io.ReadCloser, out interface{}) error {
	defer r.Close()

	if out == nil {
		return nil
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, out)
}

type Headers map[string]string
type Params map[string]interface{}
type Query map[string]interface{}

type RequestOptions struct {
	Headers Headers
	Params  Params
	Query   Query
}

func (o *RequestOptions) ContentType() string {
	return "application/x-www-form-urlencoded"
}

func (o *RequestOptions) Querystring() (string, error) {
	u, err := marshalValues(o.Query)
	if err != nil {
		return "", err
	}

	return u.Encode(), nil
}

func (o *RequestOptions) Reader() (io.Reader, error) {
	if len(o.Params) == 0 {
		return bytes.NewReader(nil), nil
	}

	u, err := marshalValues(o.Params)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader([]byte(u.Encode())), nil
}
