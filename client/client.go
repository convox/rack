package client

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/websocket"
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
	origin := fmt.Sprintf("https://%s", c.Host)
	endpoint := fmt.Sprintf("wss://%s%s", c.Host, path)

	config, err := websocket.NewConfig(endpoint, origin)

	if err != nil {
		return err
	}

	config.TlsConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	if c.Rack != "" {
		config.Header.Set("Rack", c.Rack)
	}

	config.Header.Set("Version", c.Version)

	userpass := fmt.Sprintf("convox:%s", c.Password)
	userpass_encoded := base64.StdEncoding.EncodeToString([]byte(userpass))

	config.Header.Add("Authorization", fmt.Sprintf("Basic %s", userpass_encoded))

	for k, v := range headers {
		config.Header.Add(k, v)
	}

	if c.requiresVerification() {
		config.TlsConfig = &tls.Config{
			ServerName: c.Host,
		}
	} else {
		config.TlsConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	var ws *websocket.Conn

	if proxy := os.Getenv("HTTPS_PROXY"); proxy != "" {
		ws, err = c.proxyWebsocket(config, proxy)
	} else {
		ws, err = websocket.DialConfig(config)
	}

	if err != nil {
		return err
	}

	defer ws.Close()

	var wg sync.WaitGroup

	if in != nil {
		go io.Copy(ws, in)
	}

	if out != nil {
		wg.Add(1)
		go copyAsync(out, ws, &wg)
	}

	wg.Wait()

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

func (c *Client) proxyWebsocket(config *websocket.Config, proxy string) (*websocket.Conn, error) {
	u, err := url.Parse(proxy)

	if err != nil {
		return nil, err
	}

	host := u.Host

	if !strings.Contains(host, ":") {
		host += ":443"
	}

	conn, err := net.DialTimeout("tcp", u.Host, 3*time.Second)

	if err != nil {
		return nil, err
	}

	if _, err = conn.Write([]byte(fmt.Sprintf("CONNECT %s:443 HTTP/1.1\r\n", c.Host))); err != nil {
		return nil, err
	}

	if _, err = conn.Write([]byte(fmt.Sprintf("Host: %s:443\r\n", c.Host))); err != nil {
		return nil, err
	}

	if auth := u.User; auth != nil {
		enc := base64.StdEncoding.EncodeToString([]byte(auth.String()))

		if _, err = conn.Write([]byte(fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", enc))); err != nil {
			return nil, err
		}
	}

	if _, err = conn.Write([]byte("Proxy-Connection: Keep-Alive\r\n\r\n")); err != nil {
		return nil, err
	}

	data, err := bufio.NewReader(conn).ReadString('\n')

	if err != nil {
		return nil, err
	}

	// need an http 200 response
	if !strings.Contains(string(data), " 200 ") {
		return nil, fmt.Errorf("proxy error: %s", strings.TrimSpace(string(data)))
	}

	return websocket.NewClient(config, tls.Client(conn, config.TlsConfig))
}
