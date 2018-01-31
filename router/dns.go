package router

import (
	"fmt"
	"strings"

	"github.com/miekg/dns"
)

type DNS struct {
	mux    *dns.ServeMux
	router *Router
	server *dns.Server
}

func (r *Router) NewDNS() (*DNS, error) {
	mux := dns.NewServeMux()

	d := &DNS{
		mux:    mux,
		router: r,
		server: &dns.Server{
			Addr:    fmt.Sprintf("%s:53", r.ip),
			Handler: mux,
			Net:     "udp",
		},
	}

	mux.HandleFunc(".", resolvePassthrough)

	if err := d.registerDomain(r.Domain); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *DNS) Serve() error {
	return d.server.ListenAndServe()
}

func (d *DNS) registerDomain(domain string) error {
	d.mux.HandleFunc(fmt.Sprintf("%s.", domain), d.resolveConvox)

	if err := d.setupResolver(domain); err != nil {
		return err
	}

	return nil
}

func (d *DNS) resolveConvox(w dns.ResponseWriter, r *dns.Msg) {
	m := &dns.Msg{}
	m.SetReply(r)
	m.Compress = false
	m.RecursionAvailable = true
	m.Authoritative = true

	switch r.Opcode {
	case dns.OpcodeQuery:
		for _, q := range m.Question {
			switch q.Qtype {
			case dns.TypeA:
				if ep, _ := d.router.matchEndpoint(strings.TrimSuffix(q.Name, ".")); ep != nil {
					if rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ep.IP)); err == nil {
						rr.Header().Ttl = 5
						m.Answer = append(m.Answer, rr)
						fmt.Printf("ns=convox.router at=resolve type=rack host=%q ip=%q\n", q.Name, ep.IP)
					}
				}
			}
		}
	}

	w.WriteMsg(m)
}

func resolvePassthrough(w dns.ResponseWriter, r *dns.Msg) {

	fmt.Printf("r = %+v\n", r)

	c := dns.Client{Net: "tcp"}

	rs, _, err := c.Exchange(r, "8.8.8.8:53")
	if err != nil {
		m := &dns.Msg{}
		m.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(m)
		return
	}

	w.WriteMsg(rs)

	for _, q := range r.Question {
		fmt.Printf("ns=convox.router at=resolve type=forward host=%q\n", q.Name)
	}
}
