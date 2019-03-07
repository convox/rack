package router_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/convox/rack/pkg/router"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
)

func TestDNSForward(t *testing.T) {
	r := testDNSRouter{upstream: "1.1.1.1:53"}

	testDNS(t, r, func(d *router.DNS, c testDNSResolver) {
		a, err := c.Resolve(dns.TypeA, "example.org")
		require.NoError(t, err)
		require.Equal(t, dns.RcodeSuccess, a.Rcode)
	})
}

func TestDNSResolveA(t *testing.T) {
	r := testDNSRouter{
		hosts: []string{"example.convox"},
		ip:    "1.2.3.4",
	}

	testDNS(t, r, func(d *router.DNS, c testDNSResolver) {
		a, err := c.Resolve(dns.TypeA, "example.convox")
		require.NoError(t, err)
		require.Equal(t, dns.RcodeSuccess, a.Rcode)
		require.Len(t, a.Answer, 1)
		if aa, ok := a.Answer[0].(*dns.A); ok {
			require.Equal(t, net.ParseIP("1.2.3.4").To4(), aa.A)
		} else {
			t.Fatal("invalid answer type")
		}
	})
}

func TestDNSResolveAAAA(t *testing.T) {
	r := testDNSRouter{
		hosts: []string{"example.convox"},
		ip:    "1.2.3.4",
	}

	testDNS(t, r, func(d *router.DNS, c testDNSResolver) {
		a, err := c.Resolve(dns.TypeAAAA, "example.convox")
		require.NoError(t, err)
		require.Equal(t, dns.RcodeSuccess, a.Rcode)
		require.Len(t, a.Answer, 1)
		if aa, ok := a.Answer[0].(*dns.AAAA); ok {
			require.Equal(t, net.ParseIP("1.2.3.4").To16(), aa.AAAA)
		} else {
			t.Fatal("invalid answer type")
		}
	})
}

func testDNS(t *testing.T, r testDNSRouter, fn func(d *router.DNS, c testDNSResolver)) {
	conn, err := net.ListenPacket("udp", "")
	require.NoError(t, err)

	d, err := router.NewDNS(conn, r)
	require.NoError(t, err)

	go d.ListenAndServe()

	_, port, err := net.SplitHostPort(conn.LocalAddr().String())
	require.NoError(t, err)

	c := testDNSResolver{port: port}

	fn(d, c)
}

type testDNSResolver struct {
	port string
}

func (r testDNSResolver) Resolve(q uint16, host string) (*dns.Msg, error) {
	m := &dns.Msg{}
	m.SetQuestion(fmt.Sprintf("%s.", host), q)

	c := dns.Client{}

	a, _, err := c.Exchange(m, fmt.Sprintf("127.0.0.1:%s", r.port))
	if err != nil {
		return nil, err
	}

	return a, nil
}

type testDNSRouter struct {
	hosts    []string
	ip       string
	upstream string
}

func (r testDNSRouter) ExternalIP(remote net.Addr) string {
	return r.ip
}

func (r testDNSRouter) TargetList(host string) ([]string, error) {
	for _, h := range r.hosts {
		if h == host {
			return []string{"target"}, nil
		}
	}

	return []string{}, nil
}

func (r testDNSRouter) Upstream() (string, error) {
	return r.upstream, nil
}
