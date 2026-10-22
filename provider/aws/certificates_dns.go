package aws

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

const (
	checkMissing = "missing"
	checkOK      = "ok"
	checkUnknown = "unknown"
	checkWrong   = "wrong"
)

type certificateResolver interface {
	LookupCNAME(ctx context.Context, server, name string) (string, error)
	LookupHost(ctx context.Context, server, name string) ([]string, error)
	LookupNS(ctx context.Context, server, name string) ([]*net.NS, error)
}

type netCertificateResolver struct{}

func (netCertificateResolver) resolver(server string) *net.Resolver {
	if server == "" {
		return net.DefaultResolver
	}

	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, network, net.JoinHostPort(server, "53"))
		},
	}
}

func (r netCertificateResolver) LookupCNAME(ctx context.Context, server, name string) (string, error) {
	return r.resolver(server).LookupCNAME(ctx, name)
}

func (r netCertificateResolver) LookupHost(ctx context.Context, server, name string) ([]string, error) {
	return r.resolver(server).LookupHost(ctx, name)
}

func (r netCertificateResolver) LookupNS(ctx context.Context, server, name string) ([]*net.NS, error) {
	return r.resolver(server).LookupNS(ctx, name)
}

var (
	certificateDNS             certificateResolver = netCertificateResolver{}
	certificateDNSTimeout                          = 2 * time.Second
	certificatePublicResolvers                     = []string{"1.1.1.1", "8.8.8.8"}
)

type certificateZone struct {
	name    string
	hosts   []string
	servers []string
}

type dnsChecker struct {
	authUntil time.Time

	mu     sync.Mutex
	labels map[string]*certificateZone
	vpc    bool
	zones  map[string]*certificateZone
}

func newDNSChecker(authUntil time.Time) *dnsChecker {
	return &dnsChecker{authUntil: authUntil, labels: map[string]*certificateZone{}, zones: map[string]*certificateZone{}}
}

func dnsName(name string) string {
	return strings.TrimSuffix(strings.ToLower(name), ".")
}

func dnsNotFound(err error) bool {
	var de *net.DNSError
	return errors.As(err, &de) && de.IsNotFound
}

func (d *dnsChecker) servers() []string {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.vpc {
		return []string{""}
	}

	return certificatePublicResolvers
}

func (d *dnsChecker) vpcOnly() bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.vpc
}

// zone walks up from key to the first name with its own NS records, through public resolvers so a
// private hosted zone in the VPC cannot hide the public delegation.
func (d *dnsChecker) zone(ctx context.Context, key string) *certificateZone {
	d.mu.Lock()
	z, ok := d.zones[key]
	d.mu.Unlock()

	if ok {
		return z
	}

	z = &certificateZone{}

	labels := strings.Split(key, ".")

	for i := 0; i < len(labels)-1; i++ {
		name := strings.Join(labels[i:], ".")

		d.mu.Lock()
		found, seen := d.labels[name]
		d.mu.Unlock()

		if !seen {
			found = d.zoneAt(ctx, name)

			d.mu.Lock()
			d.labels[name] = found
			d.mu.Unlock()
		}

		if found != nil {
			z = found
			break
		}
	}

	d.mu.Lock()
	d.zones[key] = z
	d.mu.Unlock()

	return z
}

// zoneAt returns the zone at name, an empty zone when no resolver answers, or nil to keep walking.
func (d *dnsChecker) zoneAt(ctx context.Context, name string) *certificateZone {
	for {
		vpc := d.vpcOnly()

		for _, server := range d.servers() {
			ns, err := d.lookupNS(ctx, server, name)
			if dnsNotFound(err) {
				return nil
			}
			if err != nil {
				continue
			}

			cname, err := d.lookupCNAME(ctx, server, name)
			if err == nil && dnsName(cname) != name {
				return nil
			}

			z := &certificateZone{name: name}

			for _, n := range ns {
				host := dnsName(n.Host)
				z.hosts = append(z.hosts, host)

				addrs, err := d.lookupHost(ctx, server, host)
				if err != nil {
					continue
				}

				for _, a := range addrs {
					if ip := net.ParseIP(a); ip != nil && ip.To4() != nil {
						z.servers = append(z.servers, a)
					}
				}
			}

			return z
		}

		if vpc || ctx.Err() != nil {
			return &certificateZone{}
		}

		d.mu.Lock()
		d.vpc = true
		d.mu.Unlock()
	}
}

func (d *dnsChecker) lookupNS(ctx context.Context, server, name string) ([]*net.NS, error) {
	ctx, cancel := context.WithTimeout(ctx, certificateDNSTimeout)
	defer cancel()

	return certificateDNS.LookupNS(ctx, server, name+".")
}

func (d *dnsChecker) lookupCNAME(ctx context.Context, server, name string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, certificateDNSTimeout)
	defer cancel()

	return certificateDNS.LookupCNAME(ctx, server, name+".")
}

func (d *dnsChecker) lookupHost(ctx context.Context, server, name string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, certificateDNSTimeout)
	defer cancel()

	return certificateDNS.LookupHost(ctx, server, name+".")
}

func (d *dnsChecker) check(ctx context.Context, z *certificateZone, name, value string) string {
	if c, ok := d.authoritative(ctx, z, name, value); ok {
		return c
	}

	return d.resolver(ctx, name, value)
}

// authoritative reports false when no nameserver answered or the run no longer asks them.
func (d *dnsChecker) authoritative(ctx context.Context, z *certificateZone, name, value string) (string, bool) {
	if d.vpcOnly() || (!d.authUntil.IsZero() && time.Now().After(d.authUntil)) {
		return "", false
	}

	name, value = dnsName(name), dnsName(value)

	for _, server := range z.servers {
		cname, err := d.lookupCNAME(ctx, server, name)
		if err == nil {
			return dnsClassify(name, cname, value), true
		}
		if dnsNotFound(err) {
			return checkMissing, true
		}
	}

	return "", false
}

func (d *dnsChecker) resolver(ctx context.Context, name, value string) string {
	name, value = dnsName(name), dnsName(value)

	cname, err := d.lookupCNAME(ctx, "", name)
	switch {
	case err == nil:
		return dnsClassify(name, cname, value)
	case dnsNotFound(err):
		return checkMissing
	default:
		return checkUnknown
	}
}

func dnsClassify(name, cname, value string) string {
	switch dnsName(cname) {
	case name:
		return checkMissing
	case value:
		return checkOK
	default:
		return checkWrong
	}
}

func (r *certificateRun) hostedZone(z *certificateZone) string {
	if z.name == "" || len(z.hosts) == 0 {
		return ""
	}

	r.mu.Lock()
	id, ok := r.hosted[z.name]
	r.mu.Unlock()

	if ok {
		return id
	}

	id, err := r.p.route53HostedZone(r.ctx, z)
	if err != nil {
		id = ""
		r.logRack("route53:"+z.name, fmt.Sprintf("route53 zone %s: %s", z.name, err))
	}

	r.mu.Lock()
	r.hosted[z.name] = id
	r.mu.Unlock()

	return id
}

// route53HostedZone returns the public hosted zone in this account that the internet uses for z:
// Route 53 gives same-named zones different nameservers, so at most one carries every public NS.
func (p *Provider) route53HostedZone(ctx context.Context, z *certificateZone) (string, error) {
	req := &route53.ListHostedZonesByNameInput{DNSName: aws.String(z.name)}

	for {
		res, err := p.route53().ListHostedZonesByNameWithContext(ctx, req)
		if err != nil {
			return "", err
		}

		for _, hz := range res.HostedZones {
			if dnsName(aws.StringValue(hz.Name)) != z.name {
				return "", nil
			}

			if hz.Config != nil && aws.BoolValue(hz.Config.PrivateZone) {
				continue
			}

			gres, err := p.route53().GetHostedZoneWithContext(ctx, &route53.GetHostedZoneInput{Id: hz.Id})
			if err != nil {
				return "", err
			}

			servers := map[string]bool{}

			if gres.DelegationSet != nil {
				for _, ns := range gres.DelegationSet.NameServers {
					servers[dnsName(aws.StringValue(ns))] = true
				}
			}

			all := true

			for _, h := range z.hosts {
				if !servers[h] {
					all = false
					break
				}
			}

			if all {
				return aws.StringValue(hz.Id), nil
			}
		}

		if !aws.BoolValue(res.IsTruncated) || dnsName(aws.StringValue(res.NextDNSName)) != z.name {
			return "", nil
		}

		req.DNSName = res.NextDNSName
		req.HostedZoneId = res.NextHostedZoneId
	}
}

func (r *certificateRun) route53Write(c *acmCertificate, zone string, rec certificateRecord) bool {
	_, err := r.p.route53().ChangeResourceRecordSetsWithContext(r.ctx, &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zone),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{{
				Action: aws.String(route53.ChangeActionCreate),
				ResourceRecordSet: &route53.ResourceRecordSet{
					Name:            aws.String(rec.Name),
					ResourceRecords: []*route53.ResourceRecord{{Value: aws.String(rec.Value)}},
					TTL:             aws.Int64(300),
					Type:            aws.String(route53.RRTypeCname),
				},
			}},
		},
	})
	if err == nil {
		return true
	}

	if awsError(err) == route53.ErrCodeInvalidChangeBatch {
		res, lerr := r.p.route53().ListResourceRecordSetsWithContext(r.ctx, &route53.ListResourceRecordSetsInput{
			HostedZoneId:    aws.String(zone),
			MaxItems:        aws.String("1"),
			StartRecordName: aws.String(rec.Name),
			StartRecordType: aws.String(route53.RRTypeCname),
		})
		if lerr == nil {
			for _, rr := range res.ResourceRecordSets {
				if dnsName(aws.StringValue(rr.Name)) != dnsName(rec.Name) || aws.StringValue(rr.Type) != route53.RRTypeCname {
					continue
				}

				if len(rr.ResourceRecords) > 0 && dnsName(aws.StringValue(rr.ResourceRecords[0].Value)) == dnsName(rec.Value) {
					return true
				}

				r.logRack("route53:"+rec.Name, fmt.Sprintf("certificate %s (%s) record %s exists with another value, left alone", c.id, c.domain, rec.Name))
				return false
			}
		}
	}

	r.logRack("route53:"+rec.Name, fmt.Sprintf("certificate %s (%s) route53 write %s: %s", c.id, c.domain, rec.Name, err))

	return false
}
