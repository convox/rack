package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	b64 "encoding/base64"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
	"golang.org/x/net/websocket"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "run",
		Description: "run a one-off command in your Convox rack",
		Usage:       "[process] [command]",
		Action:      cmdRun,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "App name. Inferred from current directory if not specified.",
			},
			cli.BoolFlag{
				Name:  "attach",
				Usage: "attach to an interactive session",
			},
		},
	})
}

func cmdRun(c *cli.Context) {
	if c.Bool("attach") {
		cmdRunAttached(c)
		return
	}

	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) < 1 {
		stdcli.Usage(c, "run")
		return
	}

	ps := c.Args()[0]

	command := ""

	if len(c.Args()) > 1 {
		args := c.Args()[1:]
		command = strings.Join(args, " ")
	}

	v := url.Values{}
	v.Set("command", command)

	_, err = ConvoxPostForm(fmt.Sprintf("/apps/%s/processes/%s/run", app, ps), v)

	if err != nil {
		stdcli.Error(err)
		return
	}

	fmt.Printf("Running %s `%s`\n", ps, command)
}

func cmdRunAttached(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	if len(c.Args()) < 2 {
		stdcli.Usage(c, "run")
		return
	}

	ps := c.Args()[0]

	host, password, err := currentLogin()

	if err != nil {
		stdcli.Error(err)
		return
	}

	origin := fmt.Sprintf("https://%s", host)
	url := fmt.Sprintf("wss://%s/apps/%s/processes/%s/run", host, app, ps)

	config, err := websocket.NewConfig(url, origin)

	if err != nil {
		stdcli.Error(err)
		return
	}

	command := strings.Join(c.Args()[1:], " ")

	config.Header.Set("Command", command)

	userpass := fmt.Sprintf("convox:%s", password)
	userpass_encoded := b64.StdEncoding.EncodeToString([]byte(userpass))

	config.Header.Add("Authorization", fmt.Sprintf("Basic %s", userpass_encoded))

	config.TlsConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	ws, err := websocket.DialConfig(config)

	if err != nil {
		stdcli.Error(err)
		return
	}

	defer ws.Close()

	var wg sync.WaitGroup

	wg.Add(2)

	stripper := CodeStripper{os.Stdout}

	go copyWait(ws, os.Stdin, &wg)
	go copyWait(stripper, ws, &wg)

	wg.Wait()
}

func copyWait(w io.Writer, r io.Reader, wg *sync.WaitGroup) {
	io.Copy(w, r)
	wg.Done()
}

var CodeRemoverRegex = regexp.MustCompile(`\x1b\[.n`)

type CodeStripper struct {
	writer io.Writer
}

func (cs CodeStripper) Write(data []byte) (int, error) {
	_, err := cs.writer.Write(CodeRemoverRegex.ReplaceAll(data, []byte("")))
	return len(data), err
}
