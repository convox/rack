package main

import (
	"crypto/tls"
	"fmt"
	"io"

	"github.com/convox/cli/Godeps/_workspace/src/golang.org/x/net/websocket"

	"github.com/convox/cli/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/convox/cli/stdcli"
)

import b64 "encoding/base64"

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "logs",
		Description: "stream the logs for an application",
		Usage:       "",
		Action:      cmdLogsStream,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "app",
				Usage: "App name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdLogsStream(c *cli.Context) {
	_, app, err := stdcli.DirApp(c, ".")

	if err != nil {
		stdcli.Error(err)
		return
	}

	host, password, err := currentLogin()

	if err != nil {
		stdcli.Error(err)
		return
	}

	origin := fmt.Sprintf("https://%s", host)
	url := fmt.Sprintf("wss://%s/apps/%s/logs", host, app)

	config, err := websocket.NewConfig(url, origin)

	if err != nil {
		stdcli.Error(err)
		return
	}

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

	for {
		var message []byte

		for {
			err := websocket.Message.Receive(ws, &message)

			if err == io.EOF {
				return
			}

			if err != nil {
				// fmt.Fprintf(os.Stderr, "ws %s, retrying...\n", err.Error())
				continue
			}

			fmt.Print(string(message))
		}
	}
}
