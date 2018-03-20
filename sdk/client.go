package sdk

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

type Client struct {
	Debug    bool
	Endpoint *url.URL
	Key      string
	Rack     string
	Socket   string
	Version  string
}

type Headers map[string]string
type Params map[string]interface{}
type Query map[string]interface{}

type RequestOptions struct {
	Body    io.Reader
	Headers Headers
	Params  Params
	Query   Query
}

func (o *RequestOptions) Querystring() (string, error) {
	u, err := marshalValues(o.Query)
	if err != nil {
		return "", err
	}

	return u.Encode(), nil
}

func (o *RequestOptions) Reader() (io.Reader, error) {
	if o.Body != nil && len(o.Params) > 0 {
		return nil, fmt.Errorf("cannot specify both Body and Params")
	}

	if o.Body == nil && len(o.Params) == 0 {
		return bytes.NewReader(nil), nil
	}

	if o.Body != nil {
		return o.Body, nil
	}

	u, err := marshalValues(o.Params)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader([]byte(u.Encode())), nil
}

func (o *RequestOptions) ContentType() string {
	if o.Body == nil {
		return "application/x-www-form-urlencoded"
	}

	return "application/octet-stream"
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

func (c *Client) Websocket(path string, opts RequestOptions) (io.ReadCloser, error) {
	var u url.URL

	u = *c.Endpoint

	u.Scheme = "wss"
	u.Path += path

	cfg, err := websocket.NewConfig(u.String(), c.Endpoint.String())
	if err != nil {
		return nil, err
	}

	cfg.TlsConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	ws, err := websocket.DialConfig(cfg)
	if err != nil {
		return nil, err
	}

	return ws, nil
}

func (c *Client) Head(path string, opts RequestOptions) error {
	req, err := c.Request("HEAD", path, opts)
	if err != nil {
		return err
	}

	_, err = c.handleRequest(req)

	return err
}

func (c *Client) Options(path string, opts RequestOptions, out interface{}) error {
	req, err := c.Request("OPTIONS", path, opts)
	if err != nil {
		return err
	}

	res, err := c.handleRequest(req)
	if err != nil {
		return err
	}

	return unmarshalReader(res.Body, out)
}

func (c *Client) GetStream(path string, opts RequestOptions) (*http.Response, error) {
	req, err := c.Request("GET", path, opts)
	if err != nil {
		return nil, err
	}

	return c.handleRequest(req)
}

func (c *Client) Get(path string, opts RequestOptions, out interface{}) error {
	res, err := c.GetStream(path, opts)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	return unmarshalReader(res.Body, out)
}

func (c *Client) PostStream(path string, opts RequestOptions) (*http.Response, error) {
	req, err := c.Request("POST", path, opts)
	if err != nil {
		return nil, err
	}

	res, err := c.handleRequest(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) Post(path string, opts RequestOptions, out interface{}) error {
	res, err := c.PostStream(path, opts)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	return unmarshalReader(res.Body, out)
}

func (c *Client) PutStream(path string, opts RequestOptions) (*http.Response, error) {
	req, err := c.Request("PUT", path, opts)
	if err != nil {
		return nil, err
	}

	return c.handleRequest(req)
}

func (c *Client) Put(path string, opts RequestOptions, out interface{}) error {
	res, err := c.PutStream(path, opts)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	return unmarshalReader(res.Body, out)
}

func (c *Client) Delete(path string, opts RequestOptions, out interface{}) error {
	req, err := c.Request("DELETE", path, opts)
	if err != nil {
		return err
	}

	res, err := c.handleRequest(req)
	if err != nil {
		return err
	}

	return unmarshalReader(res.Body, out)
}

func (c *Client) Client() *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 2 * time.Second,
	}

	t := &http.Transport{
		DialContext: func(ctx context.Context, proto, addr string) (net.Conn, error) {
			if c.Socket != "" {
				proto = "unix"
				addr = c.Socket
			}
			return dialer.DialContext(ctx, proto, addr)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// disabled because HTTP2 over ALB doesn't work yet

	// if err := http2.ConfigureTransport(t); err != nil {
	//   panic(err)
	// }

	return &http.Client{
		Transport: t,
	}
}

func (c *Client) Request(method, path string, opts RequestOptions) (*http.Request, error) {
	qs, err := opts.Querystring()
	if err != nil {
		return nil, err
	}

	r, err := opts.Reader()
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s://%s%s%s?%s", c.Endpoint.Scheme, c.Endpoint.Host, c.Endpoint.Path, path, qs)

	req, err := http.NewRequest(method, endpoint, r)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "*/*")
	req.Header.Set("Content-Type", opts.ContentType())
	req.Header.Set("User-Agent", fmt.Sprintf("convox.go/%s", c.Version))
	req.Header.Set("Version", c.Version)

	if c.Rack != "" {
		req.Header.Set("Rack", c.Rack)
	}

	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	if c.Endpoint.User != nil {
		req.SetBasicAuth("convox", c.Endpoint.User.Username())
	}

	return req, nil
}

func (c *Client) handleRequest(req *http.Request) (*http.Response, error) {
	res, err := c.Client().Do(req)
	if err != nil {
		return nil, err
	}

	if err := responseError(res); err != nil {
		return nil, err
	}

	return res, nil
}

func responseError(res *http.Response) error {
	// disabled because HTTP2 over ALB doesnt work yet

	// if !res.ProtoAtLeast(2, 0) {
	//   return fmt.Errorf("server did not respond with http/2")
	// }

	if res.StatusCode < 400 {
		return nil
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var e struct {
		Error string
	}

	if err := json.Unmarshal(data, &e); err == nil && e.Error != "" {
		return fmt.Errorf(e.Error)
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

func marshalOptions(opts interface{}) (RequestOptions, error) {
	ro := RequestOptions{
		Params: map[string]interface{}{},
		Query:  map[string]interface{}{},
	}

	v := reflect.ValueOf(opts)
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		if n := f.Tag.Get("param"); n != "" {
			if u := marshalValue(v.Field(i)); u != nil {
				ro.Params[n] = u
			}
		}

		if n := f.Tag.Get("query"); n != "" {
			if u := marshalValue(v.Field(i)); u != nil {
				ro.Query[n] = u
			}
		}
	}

	return ro, nil
}

func marshalValue(f reflect.Value) interface{} {
	if f.IsNil() {
		return nil
	}

	if f.Kind() == reflect.Ptr {
		return f.Elem().Interface()
	}

	return f.Interface()
}
