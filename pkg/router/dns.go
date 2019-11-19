package router

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/miekg/dns"
)

type DNS struct {
	internal bool
	mux      *dns.ServeMux
	router   DNSRouter
	server   *dns.Server
	upstream string
}

type DNSRouter interface {
	RouterIP(internal bool) string
	TargetList(host string) ([]string, error)
	Upstream() (string, error)
}

func NewDNS(conn net.PacketConn, router DNSRouter, internal bool) (*DNS, error) {
	mux := dns.NewServeMux()

	d := &DNS{
		internal: internal,
		mux:      mux,
		router:   router,
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
	fmt.Printf("ns=dns at=serve\n")

	return d.server.ActivateAndServe()
}

func (d *DNS) Shutdown(ctx context.Context) error {
	fmt.Printf("ns=dns at=shutdown\n")

	return d.server.Shutdown()
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

	host := strings.TrimSuffix(q.Name, ".")
	internal := d.internal

	ts, err := d.router.TargetList(host)
	if err != nil {
		dnsError(w, r, err)
		return
	}

	if len(ts) > 0 {
		fmt.Printf("ns=dns at=resolve internal=%t type=route host=%q\n", internal, q.Name)

		a := &dns.Msg{}

		if r.IsEdns0() != nil {
			a.SetEdns0(4096, true)
		}

		a.SetReply(r)

		//a.Authoritative = true
		a.Compress = false
		//a.Ns = []dns.RR{soa}
		a.RecursionAvailable = true

		ip := d.router.RouterIP(internal)

		if parts := strings.Split(host, "."); len(parts) == 2 && parts[0] == "registry" {
			switch os.Getenv("PLATFORM") {
			case "darwin":
				ip = "0.0.0.0"
			}
		}

		switch q.Qtype {
		case dns.TypeA:
			fmt.Printf("ns=dns at=answer internal=%t type=A value=%s\n", internal, ip)
			rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
			if err != nil {
				dnsError(w, r, err)
				return
			}
			a.Answer = append(a.Answer, rr)
		// case dns.TypeAAAA:
		// 	fmt.Printf("ns=dns at=answer internal=%t type=AAAA value=%s\n", internal, ip)
		// 	rr, err := dns.NewRR(fmt.Sprintf("%s AAAA %s", q.Name, ip))
		// 	if err != nil {
		// 		dnsError(w, r, err)
		// 		return
		// 	}
		// 	a.Answer = append(a.Answer, rr)
		default:
			fmt.Printf("ns=dns at=answer internal=%t type=%s value=nx\n", internal, dns.TypeToString[q.Qtype])
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

func dnsError(w dns.ResponseWriter, r *dns.Msg, err error) {
	fmt.Printf("ns=dns at=error error=%s\n", err)
	m := &dns.Msg{}
	m.SetRcode(r, dns.RcodeServerFailure)
	w.WriteMsg(m)
}
