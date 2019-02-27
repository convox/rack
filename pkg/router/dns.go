package router

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type DNS struct {
	Router   *Router
	config   *dns.ClientConfig
	mux      *dns.ServeMux
	prefix   string
	server   *dns.Server
	service  string
	upstream string
}

func NewDNS(r *Router) (*DNS, error) {
	mux := dns.NewServeMux()

	d := &DNS{
		Router: r,
		mux:    mux,
		server: &dns.Server{
			Addr:    ":5453",
			Handler: mux,
			Net:     "udp",
		},
		upstream: "8.8.8.8:53",
	}

	cc, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return nil, err
	}

	d.config = cc

	if len(d.config.Servers) > 0 {
		d.upstream = fmt.Sprintf("%s:53", d.config.Servers[0])
	}

	if host := os.Getenv("SERVICE_HOST"); host != "" {
		for {
			if ips, err := net.LookupIP(host); err == nil && len(ips) > 0 {
				d.service = ips[0].String()
				break
			}

			time.Sleep(1 * time.Second)
		}
	}

	if parts := strings.Split(os.Getenv("POD_IP"), "."); len(parts) > 2 {
		d.prefix = fmt.Sprintf("%s.%s.", parts[0], parts[1])
	}

	fmt.Printf("d.prefix: %v\n", d.prefix)
	fmt.Printf("d.service: %v\n", d.service)
	fmt.Printf("d.upstream: %v\n", d.upstream)

	mux.Handle(".", d)

	return d, nil
}

func (d *DNS) ListenAndServe() error {
	return d.server.ListenAndServe()
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

	fmt.Printf("ns=convox.router at=query remote=%s\n", w.RemoteAddr())

	q := r.Question[0]

	if d.Router.TargetCount(strings.TrimSuffix(q.Name, ".")) > 0 {
		fmt.Printf("ns=convox.router at=resolve type=route host=%q\n", q.Name)

		a := &dns.Msg{}

		if r.IsEdns0() != nil {
			a.SetEdns0(4096, true)
		}

		a.SetReply(r)

		//a.Authoritative = true
		a.Compress = false
		//a.Ns = []dns.RR{soa}
		a.RecursionAvailable = true

		ip := d.Router.IP

		if strings.HasPrefix(w.RemoteAddr().String(), d.prefix) {
			ip = d.service
		}

		switch q.Qtype {
		case dns.TypeA:
			fmt.Printf("ns=convox.router at=answer type=A value=%s\n", ip)
			rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
			if err != nil {
				dnsError(w, r, err)
				return
			}
			a.Answer = append(a.Answer, rr)
		case dns.TypeAAAA:
			fmt.Printf("ns=convox.router at=answer type=AAAA value=%s\n", ip)
			rr, err := dns.NewRR(fmt.Sprintf("%s AAAA %s", q.Name, ip))
			if err != nil {
				dnsError(w, r, err)
				return
			}
			a.Answer = append(a.Answer, rr)
		default:
			fmt.Printf("ns=convox.router at=answer type=%s value=nx\n", dns.TypeToString[q.Qtype])
		}

		w.WriteMsg(a)

		return
	}

	fmt.Printf("ns=convox.router at=resolve type=forward host=%q\n", q.Name)

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
	fmt.Printf("ns=convox.router at=error error=%s\n", err)
	m := &dns.Msg{}
	m.SetRcode(r, dns.RcodeServerFailure)
	w.WriteMsg(m)
}
