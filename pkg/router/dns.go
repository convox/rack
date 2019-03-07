package router

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
)

type DNS struct {
	mux      *dns.ServeMux
	router   DNSRouter
	server   *dns.Server
	upstream string
}

type DNSRouter interface {
	ExternalIP(remote net.Addr) string
	TargetList(host string) ([]string, error)
	Upstream() (string, error)
}

func NewDNS(conn net.PacketConn, router DNSRouter) (*DNS, error) {
	mux := dns.NewServeMux()

	d := &DNS{
		mux:    mux,
		router: router,
		server: &dns.Server{
			PacketConn: conn,
			Handler:    mux,
		},
		upstream: "1.1.1.1:53",
	}

	us, err := router.Upstream()
	if err != nil {
		return nil, err
	}

	d.upstream = us

	fmt.Printf("ns=dns at=new upstream=%s\n", d.upstream)

	mux.Handle(".", d)

	return d, nil
}

func (d *DNS) ListenAndServe() error {
	return d.server.ActivateAndServe()
}

func (d *DNS) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	//soa, err := dns.NewRR("$ORIGIN .\n$TTL 0\n@ SOA ns.convox. support.convox.com. 2018042500 0 0 0 0")
	//if err != nil {
	//	dnsError(w, r)
	//	return
	//}

	if len(r.Question) < 1 {
		dnsError(w, r, fmt.Errorf("no question"))
		return
	}

	fmt.Printf("ns=dns at=query remote=%s\n", w.RemoteAddr())

	q := r.Question[0]

	ts, err := d.router.TargetList(strings.TrimSuffix(q.Name, "."))
	if err != nil {
		dnsError(w, r, err)
		return
	}

	if len(ts) > 0 {
		fmt.Printf("ns=dns at=resolve type=route host=%q\n", q.Name)

		a := &dns.Msg{}

		if r.IsEdns0() != nil {
			a.SetEdns0(4096, true)
		}

		a.SetReply(r)

		//a.Authoritative = true
		a.Compress = false
		//a.Ns = []dns.RR{soa}
		a.RecursionAvailable = true

		ip := d.router.ExternalIP(w.RemoteAddr())

		switch q.Qtype {
		case dns.TypeA:
			fmt.Printf("ns=dns at=answer type=A value=%s\n", ip)
			rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
			if err != nil {
				dnsError(w, r, err)
				return
			}
			a.Answer = append(a.Answer, rr)
		case dns.TypeAAAA:
			fmt.Printf("ns=dns at=answer type=AAAA value=%s\n", ip)
			rr, err := dns.NewRR(fmt.Sprintf("%s AAAA %s", q.Name, ip))
			if err != nil {
				dnsError(w, r, err)
				return
			}
			a.Answer = append(a.Answer, rr)
		default:
			fmt.Printf("ns=dns at=answer type=%s value=nx\n", dns.TypeToString[q.Qtype])
		}

		w.WriteMsg(a)

		return
	}

	fmt.Printf("ns=dns at=resolve type=forward host=%q\n", q.Name)

	c := dns.Client{Net: "udp"}

	rs, _, err := c.Exchange(r, d.upstream)
	if err != nil {
		dnsError(w, r, err)
		return
	}

	//rs.Ns = []dns.RR{soa}

	w.WriteMsg(rs)
}

func (d *DNS) Shutdown(ctx context.Context) error {
	return d.server.Shutdown()
}

func dnsError(w dns.ResponseWriter, r *dns.Msg, err error) {
	fmt.Printf("ns=dns at=error error=%s\n", err)
	m := &dns.Msg{}
	m.SetRcode(r, dns.RcodeServerFailure)
	w.WriteMsg(m)
}
