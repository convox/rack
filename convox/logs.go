package main

import (
	"fmt"
	"golang.org/x/net/websocket"

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
				Name:  "name",
				Usage: "app name. Inferred from current directory if not specified.",
			},
		},
	})
}

func cmdLogsStream(c *cli.Context) {
	name := c.String("name")

	if name == "" {
		name = DirAppName()
	}

	host, password, err := currentLogin()

	if err != nil {
		stdcli.Error(err)
		return
	}

	origin := fmt.Sprintf("https://%s", host)
	url := fmt.Sprintf("ws://%s/apps/%s/logs/stream", host, name)

	config, err := websocket.NewConfig(url, origin)

	if err != nil {
		stdcli.Error(err)
		return
	}

	userpass := fmt.Sprintf("convox:%s", password)
	userpass_encoded := b64.StdEncoding.EncodeToString([]byte(userpass))

	config.Header.Add("Authorization", fmt.Sprintf("Basic %s", userpass_encoded))

	ws, err := websocket.DialConfig(config)

	if err != nil {
		stdcli.Error(err)
		return
	}

	defer ws.Close()

	var message []byte

	for {
		websocket.Message.Receive(ws, &message)
		fmt.Print(string(message))
	}
}
