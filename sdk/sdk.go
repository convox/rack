package sdk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
)

const (
	sortableTime        = "20060102.150405.000000000"
	statusCodePrefix    = "F1E49A85-0AD7-4AEF-A618-C249C6E6568D:"
	ecsExecSessionByte  = '\x00'
)

type ecsExecSession struct {
	SessionID  string `json:"sessionId"`
	StreamURL  string `json:"streamUrl"`
	TokenValue string `json:"tokenValue"`
	Region     string `json:"region"`
}

var (
	Version = "dev"
)

type AuthenticationError interface {
	AuthenticationError() error
}

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
	if err := c.Get("/racks", stdsdk.RequestOptions{}, nil); err != nil {
		if _, ok := err.(AuthenticationError); ok {
			return nil, err
		}
	}

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

func runSessionManagerPlugin(session ecsExecSession) (int, error) {
	pluginPath, err := exec.LookPath("session-manager-plugin")
	if err != nil {
		return -1, fmt.Errorf("session-manager-plugin not found in PATH. Install it: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html")
	}

	sessionJSON, err := json.Marshal(map[string]string{
		"SessionId":  session.SessionID,
		"StreamUrl":  session.StreamURL,
		"TokenValue": session.TokenValue,
	})
	if err != nil {
		return -1, err
	}

	endpoint := fmt.Sprintf("https://ssm.%s.amazonaws.com", session.Region)

	targetJSON, err := json.Marshal(map[string]string{
		"Target": session.SessionID,
	})
	if err != nil {
		return -1, err
	}

	cmd := exec.Command(pluginPath,
		string(sessionJSON),
		session.Region,
		"StartSession",
		"",
		string(targetJSON),
		endpoint,
	)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return -1, err
	}

	return 0, nil
}

func (c *Client) WithContext(ctx context.Context) structs.Provider {
	cc := *c
	cc.Client = cc.Client.WithContext(ctx)
	return &cc
}
