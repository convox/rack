package sdk

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/convox/rack/structs"
	"github.com/convox/stdsdk"
)

const (
	sortableTime     = "20060102.150405.000000000"
	statusCodePrefix = "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:"
)

type Client struct {
	*stdsdk.Client
	Debug   bool
	Rack    string
	Version string
}

// ensure interface parity
var _ structs.Provider = &Client{}

func New(endpoint string) (*Client, error) {
	s, err := stdsdk.New(coalesce(endpoint, "https://rack.convox"))
	if err != nil {
		return nil, err
	}

	c := &Client{
		Client:  s,
		Debug:   os.Getenv("CONVOX_DEBUG") == "true",
		Version: "dev",
	}

	c.Client.Headers = c.Headers

	return c, nil
}

func NewFromEnv() (*Client, error) {
	return New(os.Getenv("RACK_URL"))
}

// func (c *Client) Prepare(req *http.Request) {
//   req.Header.Set("User-Agent", fmt.Sprintf("convox.go/%s", c.Version))
//   req.Header.Set("Version", c.Version)

//   if c.Rack != "" {
//     req.Header.Set("Rack", c.Rack)
//   }
// }

func (c *Client) Headers() http.Header {
	h := http.Header{}

	// h.Set("User-Agent", fmt.Sprintf("convox.go/%s", c.Version))
	h.Set("Version", c.Version)

	fmt.Printf("c.Endpoint.User = %+v\n", c.Endpoint.User)

	if c.Endpoint.User != nil {
		h.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s", c.Endpoint.User)))))
	}

	if c.Rack != "" {
		h.Set("Rack", c.Rack)
	}

	return h
}

func (c *Client) WebsocketExit(path string, ro stdsdk.RequestOptions, rw io.ReadWriter) (int, error) {
	ws, err := c.Websocket(path, ro)
	if err != nil {
		return 0, err
	}

	buf := make([]byte, 10*1024)
	code := 0

	for {
		n, err := ws.Read(buf)
		if err == io.EOF {
			return code, nil
		}
		if err != nil {
			return code, err
		}

		if i := strings.Index(string(buf[0:n]), statusCodePrefix); i > -1 {
			if _, err := rw.Write(buf[0:i]); err != nil {
				return 0, err
			}

			m := i + len(statusCodePrefix)

			code, err = strconv.Atoi(strings.TrimSpace(string(buf[m:n])))
			if err != nil {
				fmt.Printf("err = %+v\n", err)
				return 0, fmt.Errorf("unable to read exit code")
			}

			continue
		}

		if _, err := rw.Write(buf[0:n]); err != nil {
			return 0, err
		}
	}
}
