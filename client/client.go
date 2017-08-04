package client

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	// "golang.org/x/net/websocket"
	"github.com/gorilla/websocket"
)

//this just needs to be random enough to never show up again in a byte stream
var StatusCodePrefix = "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:"

type Client struct {
	Host     string
	Password string
	Version  string

	Rack string
}

type Files map[string]io.Reader
type Params map[string]string

func New(host, password, version string) *Client {
	return &Client{
		Host:     host,
		Password: password,
		Version:  version,
	}
}

func (c *Client) Get(path string, out interface{}) error {
	req, err := c.Request("GET", path, nil)
	if err != nil {
		return err
	}

	res, err := c.client().Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if err := responseError(res); err != nil {
		return err
	}

	switch t := out.(type) {
	case []byte:
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		out = data
	case io.Writer:
		_, err := io.Copy(t, res.Body)
		return err
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, out)
}

func (c *Client) Post(path string, params Params, out interface{}) error {
	form := url.Values{}

	for k, v := range params {
		form.Set(k, v)
	}

	return c.PostBody(path, strings.NewReader(form.Encode()), out)
}

func (c *Client) PostBody(path string, body io.Reader, out interface{}) error {
	_, err := c.PostBodyResponse(path, body, out)

	return err
}

func (c *Client) PostBodyResponse(path string, body io.Reader, out interface{}) (*http.Response, error) {
	req, err := c.Request("POST", path, body)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.client().Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if err := responseError(res); err != nil {
		return nil, err
	}

	if out != nil {
		data, err := ioutil.ReadAll(res.Body)

		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(data, out)

		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

type PostMultipartOptions struct {
	Files    Files
	Params   Params
	Progress Progress
}

// PostMultipart posts a multipart message in the MIME internet format.
func (c *Client) PostMultipart(path string, opts PostMultipartOptions, out interface{}) error {

	r, w := io.Pipe()
	var pr io.Reader = r

	// Get the files size(s) before hand if any
	if opts.Progress != nil {

		var size int64
		for _, file := range opts.Files {
			switch f := file.(type) {
			// Seek() is illegal for pipded /dev/stdin so lets try to get the stat size first
			case *os.File:
				stat, err := f.Stat()
				if err != nil {
					return err
				}

				size += stat.Size()

			case io.ReadSeeker:
				s, err := io.Copy(ioutil.Discard, f)
				if err != nil {
					return err
				}
				size += s
				_, err = f.Seek(0, 0) // io.SeekStart == 0 in go1.7
				if err != nil {
					return err
				}

			default:
				size = 0 // if even one file isn't seekable, bail
				break
			}
		}

		if size > 0 {

			opts.Progress.Start(size)

			defer opts.Progress.Finish()

			pr = NewProgressReader(r, opts.Progress.Progress)
		}
	}

	writer := multipart.NewWriter(w)
	streamErr := make(chan error)
	go func() {
		var e error
		defer func() {
			writer.Close()
			w.Close()
			streamErr <- e
		}()

		for name, file := range opts.Files {
			part, err := writer.CreateFormFile(name, "binary-data")
			if err != nil {
				e = err
				return
			}

			if _, err = io.Copy(part, file); err != nil {
				e = err
				return
			}
		}

		for name, value := range opts.Params {
			writer.WriteField(name, value)
		}
		e = nil
	}()

	req, err := c.Request("POST", path, pr)
	if err != nil {
		return err
	}

	req.SetBasicAuth("convox", string(c.Password))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := c.client().Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if err := <-streamErr; err != nil {
		return err
	}

	if err := responseError(res); err != nil {
		return err
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if out != nil {
		err = json.Unmarshal(data, out)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) Put(path string, params Params, out interface{}) error {
	form := url.Values{}

	for k, v := range params {
		form.Set(k, v)
	}

	return c.PutBody(path, strings.NewReader(form.Encode()), out)
}

func (c *Client) PutBody(path string, body io.Reader, out interface{}) error {
	req, err := c.Request("PUT", path, body)

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.client().Do(req)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	if err := responseError(res); err != nil {
		return err
	}

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	return json.Unmarshal(data, out)
}

func (c *Client) Delete(path string, out interface{}) error {
	_, err := c.DeleteResponse(path, out)

	return err
}

func (c *Client) DeleteResponse(path string, out interface{}) (*http.Response, error) {
	req, err := c.Request("DELETE", path, nil)

	if err != nil {
		return nil, nil
	}

	res, err := c.client().Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if err := responseError(res); err != nil {
		return nil, err
	}

	if out != nil {
		data, err := ioutil.ReadAll(res.Body)

		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(data, out)

		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func (c *Client) Stream(path string, headers map[string]string, in io.Reader, out io.Writer) error {
	endpoint := fmt.Sprintf("wss://%s%s", c.Host, path)

	dialer := websocket.Dialer{
		Proxy: http.ProxyFromEnvironment,
	}

	header := http.Header{}

	if c.Rack != "" {
		header.Set("Rack", c.Rack)
	}

	header.Set("Version", c.Version)

	userpass := fmt.Sprintf("convox:%s", c.Password)
	userpass_encoded := base64.StdEncoding.EncodeToString([]byte(userpass))

	header.Set("Authorization", fmt.Sprintf("Basic %s", userpass_encoded))

	for k, v := range headers {
		header.Add(k, v)
	}

	if c.requiresVerification() {
		dialer.TLSClientConfig = &tls.Config{
			ServerName: c.Host,
		}
	} else {
		dialer.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	ws, _, err := dialer.Dial(endpoint, header)
	if err != nil {
		return err
	}
	defer ws.Close()

	go pingWebsocket(ws)

	if in != nil {
		go writeToWebsocket(ws, in)
	}

	if err := readFromWebsocket(out, ws); err != nil {
		return err
	}

	return nil
}

func pingWebsocket(ws *websocket.Conn) {
	tick := time.Tick(10 * time.Second)
	data := []byte{}

	for range tick {
		ws.WriteMessage(websocket.PingMessage, data)
	}
}

func readFromWebsocket(w io.Writer, ws *websocket.Conn) error {
	for {
		_, data, err := ws.ReadMessage()
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				return nil
			}
			return err
		}

		if _, err := w.Write(data); err != nil {
			return err
		}
	}

	return nil
}

func writeToWebsocket(ws *websocket.Conn, r io.Reader) error {
	data := make([]byte, 1024)

	for {
		n, err := r.Read(data)
		if err != nil {
			return err
		}

		if err := ws.WriteMessage(websocket.BinaryMessage, data[0:n]); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) requiresVerification() bool {
	return c.Host == "console.convox.com"
}

func (c *Client) client() *http.Client {
	client := &http.Client{}

	var config *tls.Config

	if c.requiresVerification() {
		config = &tls.Config{
			ServerName: c.Host,
		}
	} else {
		config = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	client.Transport = &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: config,
	}

	return client
}

func copyAsync(dst io.Writer, src io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()
	io.Copy(dst, src)
}

// Request wraps http.Request and sets some Convox-specific headers
func (c *Client) Request(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("https://%s%s", c.Host, path), body)

	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("convox", string(c.Password))

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Version", c.Version)

	if c.Rack != "" {
		req.Header.Add("Rack", c.Rack)
	}

	return req, nil
}
