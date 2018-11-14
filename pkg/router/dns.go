package router

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

type DNS struct {
	Router *Router
	mux    *dns.ServeMux
	server *dns.Server
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
	}

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

		switch q.Qtype {
		case dns.TypeA:
			rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, d.Router.IP))
			if err != nil {
				dnsError(w, r)
				return
			}
			a.Answer = append(a.Answer, rr)
		case dns.TypeAAAA:
			rr, err := dns.NewRR(fmt.Sprintf("%s AAAA %s", q.Name, d.Router.IP))
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

	c := dns.Client{Net: "tcp"}

	rs, _, err := c.Exchange(r, "8.8.8.8:53")
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
