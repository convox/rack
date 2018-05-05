package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/convox/stdcli"
)

func init() {
	CLI.Command("proxy", "proxy a connection inside the rack", Proxy, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Usage:    "<[port:]host:hostport> [[port:]host:hostport]...",
		Validate: stdcli.ArgsMin(1),
	})

}

func Proxy(c *stdcli.Context) error {
	for _, arg := range c.Args {
		parts := strings.SplitN(arg, ":", 3)

		var host string
		var port, hostport int

		switch len(parts) {
		case 2:
			host = parts[0]

			p, err := strconv.Atoi(parts[1])
			if err != nil {
				return err
			}

			port = p
			hostport = p
		case 3:
			host = parts[1]

			p, err := strconv.Atoi(parts[0])
			if err != nil {
				return err
			}

			port = p

			p, err = strconv.Atoi(parts[2])

			if err != nil {
				return err
			}

			hostport = p
		default:
			return fmt.Errorf("invalid argument: %s", arg)
		}

		go proxy(c, port, host, hostport)
	}

	// block forever
	select {}
}

func proxy(c *stdcli.Context, localport int, remotehost string, remoteport int) {
	fmt.Printf("proxying localhost:%d to %s:%d\n", localport, remotehost, remoteport)

	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", localport))
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	defer listener.Close()

	for {
		cn, err := listener.Accept()
		if err != nil {
			fmt.Printf("error: %s\n", err)
			return
		}

		fmt.Printf("connect: %d\n", localport)

		go func() {
			defer cn.Close()

			if err := provider(c).Proxy(remotehost, remoteport, cn); err != nil {
				fmt.Printf("error: %s\n", err)
			}
		}()
	}
}
