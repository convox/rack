package router

import (
	"fmt"
	"net"
	"os"
	"strings"

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

	if ips, err := net.LookupIP("router.convox-system.svc.cluster.local"); err == nil && len(ips) > 0 {
		d.service = ips[0].String()
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

func (d *DNS) Serve() error {
	return d.server.ListenAndServe()
}

func (d *DNS) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	//soa, err := dns.NewRR("$ORIGIN .\n$TTL 0\n@ SOA ns.convox. support.convox.com. 2018042500 0 0 0 0")
	//if err != nil {
	//	dnsError(w, r)
	//	return
	//}

	if len(r.Question) < 1 {
		dnsError(w, r)
		return
	}

	fmt.Printf("w.RemoteAddr: %v\n", w.RemoteAddr())

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
			rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
			if err != nil {
				dnsError(w, r)
				return
			}
			a.Answer = append(a.Answer, rr)
		case dns.TypeAAAA:
			rr, err := dns.NewRR(fmt.Sprintf("%s AAAA %s", q.Name, ip))
			if err != nil {
				dnsError(w, r)
				return
			}
			a.Answer = append(a.Answer, rr)
		}

		w.WriteMsg(a)

		return
	}

	fmt.Printf("ns=convox.router at=resolve type=forward host=%q\n", q.Name)

	c := dns.Client{Net: "udp"}

	rs, _, err := c.Exchange(r, d.upstream)
	if err != nil {
		dnsError(w, r)
		return
	}

	//rs.Ns = []dns.RR{soa}

	w.WriteMsg(rs)
}

func dnsError(w dns.ResponseWriter, r *dns.Msg) {
	m := &dns.Msg{}
	m.SetRcode(r, dns.RcodeServerFailure)
	w.WriteMsg(m)
}
