package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/convox/rack/client"
	"github.com/convox/rack/cmd/convox/stdcli"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "proxy",
		Description: "proxy local ports into a rack",
		Usage:       "<[port:]host:hostport> [[port:]host:hostport]...",
		Action:      cmdProxy,
	})
}

func cmdProxy(c *cli.Context) {
	if len(c.Args()) == 0 {
		stdcli.Usage(c, "proxy")
	}

	for _, arg := range c.Args() {
		parts := strings.SplitN(arg, ":", 3)

		var host string
		var port, hostport int

		switch len(parts) {
		case 2:
			host = parts[0]

			p, err := strconv.Atoi(parts[1])

			if err != nil {
				stdcli.Error(err)
				return
			}

			port = p
			hostport = p
		case 3:
			host = parts[1]

			p, err := strconv.Atoi(parts[0])

			if err != nil {
				stdcli.Error(err)
				return
			}

			port = p

			p, err = strconv.Atoi(parts[2])

			if err != nil {
				stdcli.Error(err)
				return
			}

			hostport = p
		default:
			stdcli.Error(fmt.Errorf("invalid argument: %s", arg))
			return
		}

		go proxy(port, host, hostport, rackClient(c))
	}

	// block forever
	select {}
}

func proxy(port int, host string, hostport int, client *client.Client) {
	fmt.Printf("proxying localhost:%d to %s:%d\n", port, host, hostport)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	defer listener.Close()

	for {
		conn, err := listener.Accept()

		if err != nil {
			fmt.Printf("error: %s\n", err)
			return
		}

		defer conn.Close()

		fmt.Printf("connect: %d\n", port)

		go func() {
			err := client.Proxy(host, hostport, conn)

			if err != nil {
				fmt.Printf("error: %s\n", err)
				conn.Close()
				return
			}
		}()
	}
}
