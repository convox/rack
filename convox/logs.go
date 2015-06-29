package main

import (
	"fmt"
	"golang.org/x/net/websocket"
	"os"
	"os/signal"

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
	})
}

func cmdLogsStream(c *cli.Context) {
	appName := DirAppName()

	host, password, err := currentLogin()

	if err != nil {
		stdcli.Error(err)
		return
	}

	origin := fmt.Sprintf("https://%s", host)
	url := fmt.Sprintf("ws://%s/apps/%s/logs/stream", host, appName)

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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	for {
		select {
		case <-quit:
			_ = ws.Close()
			return
		default:
			var message string
			websocket.Message.Receive(ws, &message)
			fmt.Print(message)
		}
	}
}
