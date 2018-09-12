package stdsdk

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	sortableTime = "20060102.150405.000000000"
)

type Client struct {
	Authenticator func(c *Client, w *http.Response) (http.Header, error)
	Endpoint      *url.URL
	Headers       HeadersFunc
}

type HeadersFunc func() http.Header

var DefaultClient = &http.Client{
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 10 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

func New(endpoint string) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	c := &Client{Endpoint: u}

	c.Headers = func() http.Header { return http.Header{} }

	return c, nil
}

func (c *Client) Head(path string, opts RequestOptions, out *bool) error {
	req, err := c.Request("HEAD", path, opts)
	if err != nil {
		return err
	}

	res, err := c.HandleRequest(req)
	if err != nil {
		return err
	}

	switch res.StatusCode / 100 {
	case 2:
		*out = true
	default:
		*out = false
	}

	return nil
}

func (c *Client) Options(path string, opts RequestOptions, out interface{}) error {
	req, err := c.Request("OPTIONS", path, opts)
	if err != nil {
		return err
	}

	res, err := c.HandleRequest(req)
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

	return c.HandleRequest(req)
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

	res, err := c.HandleRequest(req)
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

	return c.HandleRequest(req)
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

	res, err := c.HandleRequest(req)
	if err != nil {
		return err
	}

	return unmarshalReader(res.Body, out)
}

func (c *Client) Websocket(path string, opts RequestOptions) (io.ReadCloser, error) {
	var u url.URL

	u = *c.Endpoint

	u.Scheme = "wss"

	if c.Endpoint.Scheme == "http" {
		u.Scheme = "ws"
	}

	u.Path += path
	u.User = nil

	h := c.Headers()

	h.Set("Origin", fmt.Sprintf("%s://%s", c.Endpoint.Scheme, c.Endpoint.Host))

	for k, v := range opts.Headers {
		h.Set(k, v)
	}

	websocket.DefaultDialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), h)
	if err != nil {
		return nil, err
	}

	r, w := io.Pipe()

	or, ct, err := opts.Content()
	if err != nil {
		return nil, err
	}

	h.Set("Content-Type", ct)

	go websocketIn(ws, or)
	go websocketOut(w, ws)

	return r, nil
}

func websocketIn(ws *websocket.Conn, r io.Reader) {
	if r == nil {
		return
	}

	buf := make([]byte, 1024)

	for {
		n, err := r.Read(buf)
		switch err {
		case io.EOF:
			ws.WriteMessage(websocket.BinaryMessage, []byte{})
			// ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, ""))
			return
		case nil:
			ws.WriteMessage(websocket.TextMessage, buf[0:n])
		default:
			ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
			return
		}
	}
}

func websocketOut(w io.WriteCloser, ws *websocket.Conn) {
	// defer ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseGoingAway, ""))
	defer w.Close()

	for {
		code, data, err := ws.ReadMessage()
		switch err {
		case io.EOF:
			return
		case nil:
			switch code {
			case websocket.TextMessage:
				w.Write(data)
			case websocket.BinaryMessage:
				w.Close()
			}
		default:
			return
		}
	}
}

func (c *Client) Request(method, path string, opts RequestOptions) (*http.Request, error) {
	qs, err := opts.Querystring()
	if err != nil {
		return nil, err
	}

	r, ct, err := opts.Content()
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s://%s%s%s?%s", c.Endpoint.Scheme, c.Endpoint.Host, c.Endpoint.Path, path, qs)

	req, err := http.NewRequest(method, endpoint, r)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "*/*")
	req.Header.Set("Content-Type", ct)

	h := c.Headers()

	for k := range h {
		req.Header.Set(k, h.Get(k))
	}

	for k, v := range opts.Headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

func (c *Client) HandleRequest(req *http.Request) (*http.Response, error) {
	res, err := DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 401 {
		if c.Authenticator != nil {
			hs, err := c.Authenticator(c, res)
			if err != nil {
				return nil, err
			}
			if hs != nil {
				for k, v := range hs {
					for _, s := range v {
						req.Header.Add(k, s)
					}
				}
				return c.HandleRequest(req)
			}
		}
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
