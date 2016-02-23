package client

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/convox/rack/Godeps/_workspace/src/golang.org/x/net/websocket"
)

var (
	MinimumServerVersion = "20151023042141"
	StatusCodePrefix     = "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:" // needs to be random
)

type Client struct {
	Host     string
	Password string
	Version  string

	skipVersionCheck bool
}

type Params map[string]string

func New(host, password, version string) *Client {
	return &Client{
		Host:     host,
		Password: password,
		Version:  version,
	}
}

func (c *Client) Get(path string, out interface{}) error {
	err := c.versionCheck()

	if err != nil {
		return err
	}

	req, err := c.request("GET", path, nil)

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

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	err = json.Unmarshal(data, out)

	if err != nil {
		return err
	}

	return nil
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
	err := c.versionCheck()

	if err != nil {
		return nil, err
	}

	req, err := c.request("POST", path, body)

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

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, out)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) PostMultipart(path string, files map[string][]byte, params Params, out interface{}) error {
	err := c.versionCheck()

	if err != nil {
		return err
	}

	body := &bytes.Buffer{}

	writer := multipart.NewWriter(body)

	for name, source := range files {
		part, err := writer.CreateFormFile(name, "source.tgz")

		if err != nil {
			return err
		}

		_, err = io.Copy(part, bytes.NewReader(source))

		if err != nil {
			return err
		}
	}

	for name, value := range params {
		writer.WriteField(name, value)
	}

	err = writer.Close()

	if err != nil {
		return err
	}

	req, err := c.request("POST", path, body)

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

	if err := responseError(res); err != nil {
		return err
	}

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return err
	}

	err = json.Unmarshal(data, out)

	if err != nil {
		return err
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
	err := c.versionCheck()

	if err != nil {
		return err
	}

	req, err := c.request("PUT", path, body)

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

	err = json.Unmarshal(data, out)

	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Delete(path string, out interface{}) error {
	_, err := c.DeleteResponse(path, out)

	return err
}

func (c *Client) DeleteResponse(path string, out interface{}) (*http.Response, error) {
	err := c.versionCheck()

	if err != nil {
		return nil, err
	}

	req, err := c.request("DELETE", path, nil)

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

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, out)

	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *Client) Stream(path string, headers map[string]string, in io.Reader, out io.WriteCloser) error {
	origin := fmt.Sprintf("https://%s", c.Host)
	url := fmt.Sprintf("wss://%s%s", c.Host, path)

	config, err := websocket.NewConfig(url, origin)

	if err != nil {
		return err
	}

	config.TlsConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	config.Header.Set("Version", c.Version)

	userpass := fmt.Sprintf("convox:%s", c.Password)
	userpass_encoded := base64.StdEncoding.EncodeToString([]byte(userpass))

	config.Header.Add("Authorization", fmt.Sprintf("Basic %s", userpass_encoded))

	for k, v := range headers {
		config.Header.Add(k, v)
	}

	config.TlsConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	ws, err := websocket.DialConfig(config)

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

	out.Close()

	return nil
}

func (c *Client) WithoutVersionCheck(fn func(c *Client)) {
	check := c.skipVersionCheck
	c.skipVersionCheck = true
	fn(c)
	c.skipVersionCheck = check
}

func (c *Client) client() *http.Client {
	client := &http.Client{}

	client.Transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return client
}

func copyAsync(dst io.Writer, src io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()
	io.Copy(dst, src)
}

func (c *Client) request(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("https://%s%s", c.Host, path), body)

	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("convox", string(c.Password))

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Version", c.Version)

	return req, nil
}

func (c *Client) versionCheck() error {
	if c.skipVersionCheck {
		return nil
	}

	system, err := c.GetSystem()

	if err != nil {
		return err
	}

	switch v := system.Version; v {
	case "":
		return fmt.Errorf("rack outdated, please update with `convox rack update`")
	case "latest":
	default:
		if v < MinimumServerVersion {
			return fmt.Errorf("rack outdated, please update with `convox rack update`")
		}
	}

	return nil
}

func (c *Client) url(path string) string {
	return fmt.Sprintf("https://%s%s", c.Host, path)
}
