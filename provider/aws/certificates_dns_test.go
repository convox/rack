package aws

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCertificateZoneWalk(t *testing.T) {
	d := newFakeDNS()
	d.install(t)
	d.zone("convox.site", "ns-1.awsdns-01.com")
	d.zone("example.com", "ns-2.awsdns-02.net")
	d.zone("sub.example.com", "ns-3.awsdns-03.org")
	d.aliases["www.example.com"] = "example.com"

	c := newDNSChecker(time.Time{})
	ctx := context.Background()

	z := c.zone(ctx, "convox-router-1.us-test-1.convox.site")
	assert.Equal(t, "convox.site", z.name, "an apex with no address records is still the zone")
	assert.Equal(t, []string{"ns-1.awsdns-01.com"}, z.hosts)
	assert.NotEmpty(t, z.servers)

	assert.Equal(t, "example.com", c.zone(ctx, "www.example.com").name, "a label that aliases the apex is not a zone")
	assert.Equal(t, "sub.example.com", c.zone(ctx, "a.sub.example.com").name, "a delegated subzone wins over its parent")
	assert.Equal(t, "", c.zone(ctx, "host.unknown").name, "running out of labels leaves the zone unknown")

	for _, call := range d.calls {
		assert.False(t, strings.HasPrefix(call, "NS  "), "the walk uses public resolvers: %s", call)
	}
}

func TestCertificateCheck(t *testing.T) {
	d := newFakeDNS()
	d.install(t)
	d.zone("example.com", "ns-1.awsdns-01.com", "ns-2.awsdns-02.net")

	d.records["_ok.example.com"] = "_v.acm-validations.aws."
	d.records["_wrong.example.com"] = "_other.acm-validations.aws."
	d.records["_self.example.com"] = "_self.example.com."
	d.records["*.shop.example.com"] = "router.elb.amazonaws.com."

	c := newDNSChecker(time.Time{})
	ctx := context.Background()
	z := c.zone(ctx, "example.com")

	assert.Equal(t, checkOK, c.check(ctx, z, "_OK.example.com.", "_v.acm-validations.aws"), "case and trailing dots are ignored")
	assert.Equal(t, checkWrong, c.check(ctx, z, "_wrong.example.com.", "_v.acm-validations.aws."))
	assert.Equal(t, checkMissing, c.check(ctx, z, "_none.example.com.", "_v.acm-validations.aws."))
	assert.Equal(t, checkMissing, c.check(ctx, z, "_self.example.com.", "_v.acm-validations.aws."), "a name without a CNAME is missing")
	assert.Equal(t, checkWrong, c.check(ctx, z, "_x.shop.example.com.", "_v.acm-validations.aws."), "a parent wildcard answers")
}

func TestCertificateCheckNameservers(t *testing.T) {
	d := newFakeDNS()
	d.install(t)
	d.zone("example.com", "ns-1.awsdns-01.com", "ns-2.awsdns-02.net")
	d.records["_ok.example.com"] = "_v.acm-validations.aws."

	saved := certificateDNSTimeout
	certificateDNSTimeout = 20 * time.Millisecond
	t.Cleanup(func() { certificateDNSTimeout = saved })

	c := newDNSChecker(time.Time{})
	ctx := context.Background()
	z := c.zone(ctx, "example.com")

	d.refused[z.servers[0]] = true
	assert.Equal(t, checkOK, c.check(ctx, z, "_ok.example.com.", "_v.acm-validations.aws."), "REFUSED moves to the next nameserver")

	d.refused = map[string]bool{}
	d.down[z.servers[0]] = true
	d.vpc = map[string]string{}

	assert.Equal(t, checkOK, c.check(ctx, z, "_ok.example.com.", "_v.acm-validations.aws."), "a timeout moves to the next nameserver")
	assert.False(t, c.vpcOnly(), "one slow nameserver does not move the run to the VPC resolver")

	d.down[z.servers[1]] = true
	d.vpc["_ok.example.com"] = "_v.acm-validations.aws."
	assert.Equal(t, checkOK, c.check(ctx, z, "_ok.example.com.", "_v.acm-validations.aws."), "the VPC resolver answers when no nameserver does")

	delete(d.vpc, "_ok.example.com")
	assert.Equal(t, checkMissing, c.check(ctx, z, "_ok.example.com.", "_v.acm-validations.aws."))

	d.vpc = nil
	d.hang = true
	assert.Equal(t, checkUnknown, c.check(ctx, z, "_ok.example.com.", "_v.acm-validations.aws."))
}

func TestCertificateCheckVPCOnly(t *testing.T) {
	d := newFakeDNS()
	d.install(t)
	d.zone("example.com", "ns-1.awsdns-01.com")
	d.down["1.1.1.1"] = true
	d.down["8.8.8.8"] = true
	d.vpc = map[string]string{"_ok.example.com": "_v.acm-validations.aws."}

	saved := certificateDNSTimeout
	certificateDNSTimeout = 20 * time.Millisecond
	t.Cleanup(func() { certificateDNSTimeout = saved })

	c := newDNSChecker(time.Time{})
	ctx := context.Background()

	z := c.zone(ctx, "example.com")
	assert.Equal(t, "example.com", z.name, "the walk falls back to the VPC resolver")
	assert.True(t, c.vpcOnly())

	d.calls = nil

	assert.Equal(t, checkOK, c.check(ctx, z, "_ok.example.com.", "_v.acm-validations.aws."))
	assert.Equal(t, []string{"CNAME  _ok.example.com"}, d.calls, "no authoritative query once public DNS is unreachable")

	late := newDNSChecker(time.Now().Add(-time.Second))
	d.down = map[string]bool{}
	z = late.zone(ctx, "example.com")
	d.calls = nil

	assert.Equal(t, checkOK, late.check(ctx, z, "_ok.example.com.", "_v.acm-validations.aws."))
	assert.Equal(t, []string{"CNAME  _ok.example.com"}, d.calls, "past the cutoff only the VPC resolver is asked")
}
