package router

import (
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
)

type DNS struct {
	ip     net.IP
	lookup DNSLookup
	mux    *dns.ServeMux
	router *Router
	server *dns.Server
}

type DNSLookup func(string) net.IP

func NewDNS(ip net.IP, lookup DNSLookup) (*DNS, error) {
	mux := dns.NewServeMux()

	d := &DNS{
		ip:     ip,
		lookup: lookup,
		mux:    mux,
		server: &dns.Server{
			Addr:    fmt.Sprintf("%s:53", ip),
			Handler: mux,
			Net:     "udp",
		},
	}

	mux.HandleFunc(".", resolvePassthrough)

	return d, nil
}

func (d *DNS) Serve() error {
	return d.server.ListenAndServe()
}

func (d *DNS) registerDomain(domain string) error {
	fn, err := d.resolveHost(domain)
	if err != nil {
		return err
	}

	d.mux.HandleFunc(fmt.Sprintf("%s.", domain), fn)

	if err := d.setupResolver(domain, d.ip); err != nil {
		return err
	}

	return nil
}

func (d *DNS) unregisterDomain(domain string) error {
	d.mux.HandleRemove(fmt.Sprintf("%s.", domain))
	return nil
}

func (d *DNS) resolveHost(domain string) (dns.HandlerFunc, error) {
	soa, err := dns.NewRR(fmt.Sprintf("$ORIGIN %s.\n$TTL 0\n@ SOA ns.convox. support.convox.com. 2018042500 0 0 0 0", domain))
	if err != nil {
		return nil, err
	}

	fn := func(w dns.ResponseWriter, r *dns.Msg) {
		m := &dns.Msg{}
		m.SetReply(r)
		m.Ns = []dns.RR{soa}
		m.Compress = false
		m.RecursionAvailable = true
		m.Authoritative = true

		switch r.Opcode {
		case dns.OpcodeQuery:
			for _, q := range m.Question {
				switch q.Qtype {
				case dns.TypeA:
					if ip := d.lookup(strings.TrimSuffix(q.Name, ".")); ip != nil {
						if rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip)); err == nil {
							rr.Header().Ttl = 5
							m.Answer = append(m.Answer, rr)
							fmt.Printf("ns=convox.router at=resolve type=rack host=%q ip=%q\n", q.Name, ip)
						}
					} else {
						fmt.Printf("ns=convox.router at=resolve type=rack host=%q ip=nxdomain\n", q.Name)
						m.MsgHdr.Rcode = dns.RcodeNameError
					}
				}
			}
		}

		w.WriteMsg(m)
	}

	return fn, nil
}

func resolvePassthrough(w dns.ResponseWriter, r *dns.Msg) {
	soa, _ := dns.NewRR("$ORIGIN .\n$TTL 0\n@ SOA ns.convox. support.convox.com. 2018042500 0 0 0 0")

	c := dns.Client{Net: "tcp"}

	rs, _, err := c.Exchange(r, "8.8.8.8:53")
	if err != nil {
		m := &dns.Msg{}
		m.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(m)
		return
	}

	rs.Ns = []dns.RR{soa}

	w.WriteMsg(rs)

	for _, q := range r.Question {
		fmt.Printf("ns=convox.router at=resolve type=forward host=%q\n", q.Name)
	}
}
