package sdk

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
)

const (
	sortableTime     = "20060102.150405.000000000"
	statusCodePrefix = "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:"
)

var (
	Version = "dev"
)

type Client struct {
	*stdsdk.Client
	Debug   bool
	Rack    string
	Session SessionFunc
}

type SessionFunc func(c *Client) string

// ensure interface parity
var _ structs.Provider = &Client{}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func New(endpoint string) (*Client, error) {
	s, err := stdsdk.New(coalesce(endpoint, "https://rack.convox"))
	if err != nil {
		return nil, err
	}

	c := &Client{
		Client: s,
		Debug:  os.Getenv("CONVOX_DEBUG") == "true",
	}

	c.Client.Headers = c.Headers

	return c, nil
}

func NewFromEnv() (*Client, error) {
	return New(os.Getenv("RACK_URL"))
}

func (c *Client) Headers() http.Header {
	h := http.Header{}

	h.Set("User-Agent", fmt.Sprintf("convox.go/%s", Version))
	h.Set("Version", Version)

	if c.Endpoint.User != nil {
		h.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s", c.Endpoint.User)))))
	}

	if c.Rack != "" {
		h.Set("Rack", c.Rack)
	}

	if c.Session != nil {
		h.Set("Session", c.Session(c))
	}

	return h
}

func (c *Client) Websocket(path string, opts stdsdk.RequestOptions) (io.ReadCloser, error) {
	// trigger session authentication
	c.Get("/racks", stdsdk.RequestOptions{}, nil)

	return c.Client.Websocket(path, opts)
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
				return 0, fmt.Errorf("unable to read exit code")
			}

			continue
		}

		if _, err := rw.Write(buf[0:n]); err != nil {
			return 0, err
		}
	}
}

func (c *Client) WithContext(ctx context.Context) structs.Provider {
	cc := *c
	cc.Client = cc.Client.WithContext(ctx)
	return &cc
}
