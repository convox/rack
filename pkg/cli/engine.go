package cli

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/convox/rack/pkg/token"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
	"github.com/convox/stdsdk"
)

type Engine struct {
	*stdcli.Engine
	Client sdk.Interface
}

func (e *Engine) Command(command, description string, fn HandlerFunc, opts stdcli.CommandOptions) {
	wfn := func(c *stdcli.Context) error {
		return fn(e.currentClient(c), c)
	}

	e.Engine.Command(command, description, wfn, opts)
}

func (e *Engine) CommandWithoutProvider(command, description string, fn HandlerFunc, opts stdcli.CommandOptions) {
	wfn := func(c *stdcli.Context) error {
		return fn(nil, c)
	}

	e.Engine.Command(command, description, wfn, opts)
}

func (e *Engine) RegisterCommands() {
	for _, c := range commands {
		if c.Rack {
			e.Command(c.Command, c.Description, c.Handler, c.Opts)
		} else {
			e.CommandWithoutProvider(c.Command, c.Description, c.Handler, c.Opts)
		}
	}
}

var reSessionAuthentication = regexp.MustCompile(`^Session path="([^"]+)" token="([^"]+)"$`)

type session struct {
	Id string `json:"id"`
}

func (e *Engine) authenticator(c *stdsdk.Client, res *http.Response) (http.Header, error) {
	m := reSessionAuthentication.FindStringSubmatch(res.Header.Get("WWW-Authenticate"))
	if len(m) < 3 {
		return nil, nil
	}

	body := []byte{}
	headers := map[string]string{}

	if m[2] == "true" {
		areq, err := c.GetStream(m[1], stdsdk.RequestOptions{})
		if err != nil {
			return nil, err
		}
		defer areq.Body.Close()

		dreq, err := ioutil.ReadAll(areq.Body)
		if err != nil {
			return nil, err
		}

		e.Writer.Writef("Waiting for security token... ")

		data, err := token.Authenticate(dreq)
		if err != nil {
			return nil, err
		}

		e.Writer.Writef("<ok>OK</ok>\n")

		body = data
		headers["Challenge"] = areq.Header.Get("Challenge")
	}

	var s session

	ro := stdsdk.RequestOptions{
		Body:    bytes.NewReader(body),
		Headers: stdsdk.Headers(headers),
	}

	if err := c.Post(m[1], ro, &s); err != nil {
		return nil, err
	}

	h := http.Header{}

	h.Set("Session", s.Id)

	return h, nil
}

func (e *Engine) currentClient(c *stdcli.Context) sdk.Interface {
	if e.Client != nil {
		return e.Client
	}

	host, err := currentHost(c)
	if err != nil {
		c.Fail(err)
	}

	r := currentRack(c, host)

	endpoint, err := currentEndpoint(c, r)
	if err != nil {
		c.Fail(err)
	}

	sc, err := sdk.New(endpoint)
	if err != nil {
		c.Fail(err)
	}

	sc.Authenticator = e.authenticator
	sc.Rack = r

	return sc
}

var commands = []command{}

type command struct {
	Command     string
	Description string
	Handler     HandlerFunc
	Opts        stdcli.CommandOptions
	Rack        bool
}

func register(cmd, description string, fn HandlerFunc, opts stdcli.CommandOptions) {
	commands = append(commands, command{
		Command:     cmd,
		Description: description,
		Handler:     fn,
		Opts:        opts,
		Rack:        true,
	})
}

func registerWithoutProvider(cmd, description string, fn HandlerFunc, opts stdcli.CommandOptions) {
	commands = append(commands, command{
		Command:     cmd,
		Description: description,
		Handler:     fn,
		Opts:        opts,
		Rack:        false,
	})
}
