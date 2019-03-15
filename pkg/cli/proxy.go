package cli

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("proxy", "proxy a connection inside the rack", Proxy, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			stdcli.BoolFlag("tls", "t", "wrap connection in tls"),
		},
		Usage:    "<[port:]host:hostport> [[port:]host:hostport]...",
		Validate: stdcli.ArgsMin(1),
	})
}

var ProxyCloser = make(chan error)

func Proxy(rack sdk.Interface, c *stdcli.Context) error {
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

		go proxy(rack, c, port, host, hostport, c.Bool("tls"))
	}

	// block until something sends data on this channel
	return <-ProxyCloser
}

func proxy(rack sdk.Interface, c *stdcli.Context, localport int, remotehost string, remoteport int, secure bool) {
	c.Writef("proxying localhost:%d to %s:%d\n", localport, remotehost, remoteport)

	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", localport))
	if err != nil {
		c.Error(err)
		return
	}

	defer listener.Close()

	for {
		cn, err := listener.Accept()
		if err != nil {
			c.Error(err)
			return
		}

		c.Writef("connect: %d\n", localport)

		go func() {
			defer cn.Close()

			opts := structs.ProxyOptions{
				TLS: options.Bool(secure),
			}

			if err := rack.Proxy(remotehost, remoteport, cn, opts); err != nil {
				c.Error(err)
			}
		}()
	}
}
