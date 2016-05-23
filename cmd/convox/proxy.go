package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"gopkg.in/urfave/cli.v1"
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

		go proxy("127.0.0.1", port, host, hostport, rackClient(c))
	}

	// block forever
	select {}
}

func proxy(localhost string, localport int, remotehost string, remoteport int, client *client.Client) {
	fmt.Printf("proxying %s:%d to %s:%d\n", localhost, localport, remotehost, remoteport)

	listener, err := net.Listen("tcp4", fmt.Sprintf("%s:%d", localhost, localport))

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

		fmt.Printf("connect: %d\n", localport)

		go func() {
			err := client.Proxy(remotehost, remoteport, conn)

			if err != nil {
				fmt.Printf("error: %s\n", err)
				conn.Close()
				return
			}
		}()
	}
}
