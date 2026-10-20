package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	arnApi    = "arn:aws:acm:us-test-1:123456789012:certificate/11111111-0000-0000-0000-a00000000001"
	arnRouter = "arn:aws:acm:us-test-1:123456789012:certificate/11111111-0000-0000-0000-a00000000002"
	arnSvc    = "arn:aws:acm:us-test-1:123456789012:certificate/11111111-0000-0000-0000-a00000000003"
	arnApex   = "arn:aws:acm:us-test-1:123456789012:certificate/22222222-0000-0000-0000-b00000000001"
	arnWild   = "arn:aws:acm:us-test-1:123456789012:certificate/22222222-0000-0000-0000-b00000000002"
	arnOther  = "arn:aws:acm:us-test-1:123456789012:certificate/33333333-0000-0000-0000-c00000000001"
	arnGen1   = "arn:aws:acm:us-test-1:123456789012:certificate/44444444-0000-0000-0000-d00000000001"
	arnNLB    = "arn:aws:acm:us-test-1:123456789012:certificate/55555555-0000-0000-0000-e00000000001"
	arnImport = "arn:aws:acm:us-test-1:123456789012:certificate/66666666-0000-0000-0000-f00000000001"
	routerLB  = "arn:aws:elasticloadbalancing:us-test-1:123456789012:loadbalancer/app/convox-Router/0123456789abcdef"
	nlbLB     = "arn:aws:elasticloadbalancing:us-test-1:123456789012:loadbalancer/net/convox-NLB/0123456789abcdef"

	siteRouter = "convox-router-123456789.us-test-1.convox.site"
)

type certificateFixture struct {
	aws *fakeAWS
	dns *fakeDNS
	p   *Provider
	reg *fakeRegistrar
}

func newCertificateFixture(t *testing.T) *certificateFixture {
	f := &certificateFixture{aws: newFakeAWS(t), dns: newFakeDNS(), reg: &fakeRegistrar{}}
	f.reg.dns = f.dns

	f.dns.zone("convox.site", "ns-1.awsdns-01.com", "ns-2.awsdns-02.net")
	f.dns.zone("example.com", "ns-3.awsdns-03.org", "ns-4.awsdns-04.co.uk")
	f.dns.records["*."+siteRouter] = "convox-router-123456789.us-test-1.elb.amazonaws.com."

	f.dns.install(t)
	f.reg.install(t)

	f.aws.resources = []fakeResource{
		{name: "ApiBalancerCertificate", kind: "AWS::CertificateManager::Certificate", id: arnApi},
		{name: "RouterApiCertificate", kind: "AWS::CertificateManager::Certificate", id: arnRouter},
		{name: "Router", kind: "AWS::ElasticLoadBalancingV2::LoadBalancer", id: routerLB},
		{name: "NLBRouter", kind: "AWS::ElasticLoadBalancingV2::LoadBalancer", id: nlbLB},
	}

	f.aws.stacks = []fakeStack{
		{name: "convox-web", tags: map[string]string{"System": "convox", "Type": "app", "Rack": "convox", "Name": "web", "Generation": "2"}},
		{name: "convox-old", params: map[string]string{"WebPort443Listener": "32410," + arnGen1}, tags: map[string]string{"System": "convox", "Type": "app", "Rack": "convox", "Name": "old"}},
	}

	f.p = f.aws.provider()

	fastCertificates(t)

	return f
}

func fastCertificates(t *testing.T) {
	saved := []*time.Duration{&certificateRecordTick, &certificateRecordWait, &certificateGenerateWait, &certificateDNSTimeout, &acmPaceInterval}
	values := []time.Duration{}

	for _, d := range saved {
		values = append(values, *d)
	}

	certificateRecordTick = time.Millisecond
	certificateRecordWait = time.Second
	certificateGenerateWait = time.Second
	certificateDNSTimeout = 50 * time.Millisecond
	acmPaceInterval = 0

	t.Cleanup(func() {
		for i, d := range saved {
			*d = values[i]
		}
	})
}

func (f *certificateFixture) add(c *fakeCert) *fakeCert {
	if c.kind == "" {
		c.kind = "AMAZON_ISSUED"
	}
	if c.status == "" {
		c.status = "ISSUED"
	}
	if c.tags == nil {
		c.tags = map[string]string{}
	}
	if c.created.IsZero() {
		c.created = time.Now().Add(-24 * time.Hour)
	}
	if c.notAfter.IsZero() && c.status == "ISSUED" {
		c.notAfter = time.Now().Add(90 * 24 * time.Hour)
	}

	f.aws.certs = append(f.aws.certs, c)

	return c
}

func (f *certificateFixture) publish(key string) {
	rec := fakeRecord(key)
	f.dns.set(rec.Name, rec.Value)
}

func migrations(ms structs.CertificateMigrations) map[string]structs.CertificateMigration {
	out := map[string]structs.CertificateMigration{}

	for _, m := range ms {
		out[m.Id] = m
	}

	return out
}

func lookups(f *fakeAWS) []map[string]interface{} {
	out := []map[string]interface{}{}

	for _, r := range f.requests {
		if r["op"] == "RequestCertificate" {
			out = append(out, r["body"].(map[string]interface{}))
		}
	}

	return out
}

func TestCertificatePassConvoxSite(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnRouter, domain: "*." + siteRouter, email: true})
	f.add(&fakeCert{arn: arnSvc, domain: "*." + siteRouter, sans: []string{"*.web-api." + siteRouter}, email: true, tags: map[string]string{"System": "convox", "Type": "app", "Rack": "convox", "Name": "web"}})

	ms, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	got := migrations(ms)
	assert.Equal(t, resultPublishing, got[acmCertificateId(arnRouter)].Result)
	assert.Equal(t, resultPublishing, got[acmCertificateId(arnSvc)].Result)

	require.Len(t, f.reg.requests, 2)
	assert.Equal(t, "*."+siteRouter, f.reg.requests[0].Domain)
	assert.Equal(t, fakeRecord(siteRouter).Name, f.reg.requests[0].Name)
	assert.Equal(t, "*.web-api."+siteRouter, f.reg.requests[1].Domain)
	assert.Equal(t, fakeRecord("web-api."+siteRouter).Name, f.reg.requests[1].Name)
	assert.Equal(t, 0, f.aws.count("UpdateCertificateOptions"))

	require.Len(t, lookups(f.aws), 2, "one lookup per certificate")
	assert.Empty(t, f.aws.cert(arnRouter).tags["Purpose"])
	for _, c := range f.aws.certs {
		assert.NotEqual(t, lookupPurpose, c.tags["Purpose"], "lookups are deleted")
	}

	ms, err = f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	got = migrations(ms)
	assert.Equal(t, resultSwitching, got[acmCertificateId(arnRouter)].Result)
	assert.Equal(t, resultSwitching, got[acmCertificateId(arnSvc)].Result)
	assert.Equal(t, 2, f.aws.count("UpdateCertificateOptions"))
	assert.Len(t, lookups(f.aws), 2, "cached records need no new lookup")
	assert.Len(t, f.reg.requests, 2, "a resolving record is not sent again")
}

func TestCertificateLookupRequest(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", sans: []string{"example.com", "*.example.com", "www.example.com"}, email: true, tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnWild, domain: "*.shop.example.com", email: true, tags: map[string]string{"Rack": "convox"}})

	_, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	reqs := lookups(f.aws)
	require.Len(t, reqs, 2)

	byDomain := map[string]map[string]interface{}{}
	for _, r := range reqs {
		byDomain[r["DomainName"].(string)] = r
	}

	apex := byDomain["example.com"]
	require.NotNil(t, apex)
	assert.Regexp(t, `^[A-Za-z]{32}$`, apex["IdempotencyToken"])
	assert.Equal(t, "DNS", apex["ValidationMethod"])
	assert.Equal(t, []interface{}{"www.example.com"}, apex["SubjectAlternativeNames"], "the DomainName and names sharing its record are left out")
	assert.ElementsMatch(t, []interface{}{
		map[string]interface{}{"Key": "System", "Value": "convox"},
		map[string]interface{}{"Key": "Rack", "Value": "convox"},
		map[string]interface{}{"Key": "Purpose", "Value": lookupPurpose},
	}, apex["Tags"])

	wild := byDomain["*.shop.example.com"]
	require.NotNil(t, wild)
	assert.Nil(t, wild["SubjectAlternativeNames"])

	var cache certificateCache
	require.NoError(t, json.Unmarshal(f.aws.s3[certificateCacheKey], &cache))
	for _, k := range []string{"example.com", "www.example.com", "shop.example.com"} {
		assert.Equal(t, fakeRecord(k), cache.Records[k])
	}

	assert.Len(t, f.aws.certs, 2, "only the lookups were deleted")
	assert.Equal(t, 2, f.aws.count("DeleteCertificate"))
}

func TestCertificateLookupSeededAndDeferred(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnOther, domain: "example.com"})
	f.add(&fakeCert{arn: arnApex, domain: "*.example.com", email: true, tags: map[string]string{"Rack": "convox"}})

	ms, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	assert.Empty(t, lookups(f.aws), "a record seen on a DNS certificate needs no lookup")
	assert.Equal(t, resultWaiting, migrations(ms)[acmCertificateId(arnApex)].Result)

	g := newCertificateFixture(t)
	g.add(&fakeCert{arn: arnWild, domain: "shop.example.com", email: true, tags: map[string]string{"Rack": "convox"}})

	ms, err = g.p.certificatePass(nil, time.Now().Add(10*time.Second), false)
	require.NoError(t, err)
	assert.Empty(t, lookups(g.aws), "no lookup starts near the API deadline")
	assert.Equal(t, resultPreparing, migrations(ms)[acmCertificateId(arnWild)].Result)
}

func TestCertificateLookupFailure(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnWild, domain: "shop.example.com", email: true, tags: map[string]string{"Rack": "convox"}})
	f.aws.errors["RequestCertificate"] = "LimitExceededException"

	ms, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	m := migrations(ms)[acmCertificateId(arnWild)]
	assert.Equal(t, resultError, m.Result)
	assert.Contains(t, m.Detail, "LimitExceededException")

	dns, err := f.p.CertificateDns()
	require.NoError(t, err)
	require.Len(t, dns.Errors, 1)
	assert.Equal(t, acmCertificateId(arnWild), dns.Errors[0].Id)
	assert.Empty(t, dns.Preparing)
}

func TestCertificateLookupDeletedAfterDeadline(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnWild, domain: "shop.example.com", email: true, tags: map[string]string{"Rack": "convox"}})

	certificateRecordWait = 20 * time.Millisecond
	f.aws.lookupBare = true

	ms, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	assert.Equal(t, resultError, migrations(ms)[acmCertificateId(arnWild)].Result)
	assert.Equal(t, 1, f.aws.count("DeleteCertificate"))
	assert.Len(t, f.aws.certs, 1)
}

func TestCertificateSweep(t *testing.T) {
	f := newCertificateFixture(t)

	lookup := map[string]string{"System": "convox", "Rack": "convox", "Purpose": lookupPurpose}

	f.add(&fakeCert{arn: arnApex, domain: "old.example.com", status: "PENDING_VALIDATION", tags: lookup, created: time.Now().Add(-time.Hour)})
	f.add(&fakeCert{arn: arnWild, domain: "new.example.com", status: "PENDING_VALIDATION", tags: lookup, created: time.Now()})
	f.add(&fakeCert{arn: arnOther, domain: "timed.example.com", status: "VALIDATION_TIMED_OUT", tags: lookup, created: time.Now().Add(-100 * time.Hour)})
	f.add(&fakeCert{arn: arnNLB, domain: "used.example.com", tags: lookup, inUse: []string{nlbLB}})
	f.add(&fakeCert{arn: arnGen1, domain: "other.example.com", status: "PENDING_VALIDATION", tags: map[string]string{"Rack": "other", "Purpose": lookupPurpose}})

	_, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	left := []string{}
	for _, c := range f.aws.certs {
		left = append(left, c.arn)
	}

	assert.ElementsMatch(t, []string{arnWild, arnNLB, arnGen1}, left)

	cs, err := f.p.CertificateList()
	require.NoError(t, err)
	assert.Empty(t, cs, "lookups are never listed")

	rcs, err := f.p.releaseCertificates()
	require.NoError(t, err)
	assert.Empty(t, rcs)
}

func TestCertificateOwnership(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApi, domain: "*.convox-api-1.us-test-1.convox.site"})
	f.add(&fakeCert{arn: arnSvc, domain: "*." + siteRouter, tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnOther, domain: "*.other-router-1.us-test-1.convox.site"})
	f.add(&fakeCert{arn: arnApex, domain: "a.example.com", tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnWild, domain: "b.example.com", inUse: []string{routerLB}})
	f.add(&fakeCert{arn: arnNLB, domain: "c.example.com", inUse: []string{nlbLB}})
	f.add(&fakeCert{arn: arnGen1, domain: "d.example.com"})
	f.add(&fakeCert{arn: arnRouter, domain: "e.example.com", tags: map[string]string{"Rack": "other"}, inUse: []string{"arn:aws:elasticloadbalancing:us-test-1:123456789012:loadbalancer/app/other/1"}})
	f.add(&fakeCert{arn: arnImport, domain: "f.example.com", kind: "IMPORTED", tags: map[string]string{"Rack": "convox"}})

	cs, err := f.p.CertificateList()
	require.NoError(t, err)

	owned := map[string]bool{}
	for _, c := range cs {
		owned[c.Arn] = c.Owned
	}

	assert.Equal(t, map[string]bool{
		arnApi: true, arnSvc: true, arnOther: false, arnApex: true, arnWild: true, arnNLB: true, arnGen1: true, arnRouter: false,
		arnImport: false,
	}, owned)
}

func TestCertificateListStatus(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "dns.example.com"})
	f.add(&fakeCert{arn: arnWild, domain: "email.example.com", email: true})
	f.add(&fakeCert{arn: arnOther, domain: "migrating.example.com", email: true, migrating: true})
	f.add(&fakeCert{arn: arnGen1, domain: "pending.example.com", status: "PENDING_VALIDATION"})
	f.add(&fakeCert{arn: arnNLB, domain: "failed.example.com", status: "FAILED"})
	f.add(&fakeCert{arn: arnApi, domain: "imported.example.com", kind: "IMPORTED"})
	f.add(&fakeCert{arn: arnRouter, domain: "expired.example.com", status: "EXPIRED"})
	f.add(&fakeCert{arn: arnSvc, domain: "app.example.com", email: true, tags: map[string]string{"System": "convox", "Type": "app", "Rack": "convox"}})

	cs, err := f.p.CertificateList()
	require.NoError(t, err)

	status := map[string]string{}
	for _, c := range cs {
		status[c.Domain] = c.Status
	}

	assert.Equal(t, map[string]string{
		"dns.example.com":       statusDNS,
		"email.example.com":     statusEmail,
		"migrating.example.com": statusMigrating,
		"pending.example.com":   statusPending,
		"failed.example.com":    statusFailed,
		"imported.example.com":  statusImported,
		"app.example.com":       statusEmail,
	}, status)

	f.aws.lcdvError = "AccessDeniedException"

	cs, err = f.p.CertificateList()
	require.NoError(t, err, "a validations error never breaks the listing")

	for _, c := range cs {
		if c.Domain == "migrating.example.com" {
			assert.Equal(t, statusEmail, c.Status)
		}
	}
}

func TestCertificateListKeyTypesAll(t *testing.T) {
	f := newCertificateFixture(t)

	_, err := f.p.CertificateList()
	require.NoError(t, err)

	for _, r := range f.aws.requests {
		if r["op"] == "ListCertificates" {
			assert.Equal(t, acmAllKeyTypes, r["raw"])
		}
	}
}

func TestCertificateListStaleDescribe(t *testing.T) {
	f := newCertificateFixture(t)

	c := f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, switched: true})

	cert := &acmCertificate{arn: c.arn, id: acmCertificateId(c.arn), status: "ISSUED", kind: "AMAZON_ISSUED", tags: map[string]string{}}

	require.NoError(t, f.p.acmCertificateRead(f.p.Context(), cert, true))
	assert.Equal(t, statusDNS, cert.statusValue())
}

func TestCertificateCustomDomainSwitch(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnWild, domain: "*.example.com", email: true, inUse: []string{routerLB}})

	ms, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	got := migrations(ms)
	assert.Equal(t, resultWaiting, got[acmCertificateId(arnApex)].Result)
	assert.Equal(t, resultWaiting, got[acmCertificateId(arnWild)].Result)
	assert.Len(t, lookups(f.aws), 1, "the second certificate shares the first one's record")
	assert.Empty(t, f.reg.requests)

	dns, err := f.p.CertificateDns()
	require.NoError(t, err)
	require.Len(t, dns.Records, 1)
	assert.Equal(t, "example.com", dns.Records[0].Zone)
	assert.Equal(t, checkMissing, dns.Records[0].Check)
	assert.Equal(t, []string{acmCertificateId(arnApex), acmCertificateId(arnWild)}, dns.Records[0].Certificates)

	f.publish("example.com")

	ms, err = f.p.CertificateMigrate([]string{acmCertificateId(arnApex)})
	require.NoError(t, err)
	require.Len(t, ms, 1)
	assert.Equal(t, resultSwitching, ms[0].Result)
	assert.Equal(t, 1, f.aws.count("UpdateCertificateOptions"))

	ms, err = f.p.CertificateMigrationList([]string{acmCertificateId(arnApex), acmCertificateId(arnWild)})
	require.NoError(t, err)
	got = migrations(ms)
	assert.Equal(t, resultSwitching, got[acmCertificateId(arnApex)].Result, "a running migration whose records resolve")
	assert.Equal(t, resultSwitching, got[acmCertificateId(arnWild)].Result, "the worker switches it on its next pass")
	assert.Equal(t, 1, f.aws.count("UpdateCertificateOptions"), "the read-only route never switches")
}

func TestCertificateMigrateNamed(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnOther, domain: "other.example.com", email: true})
	f.add(&fakeCert{arn: arnApi, domain: "imported.example.com", kind: "IMPORTED"})

	ms, err := f.p.CertificateMigrate([]string{acmCertificateId(arnOther), acmCertificateId(arnApi), "acm-000000000000"})
	require.NoError(t, err)

	got := migrations(ms)
	assert.Equal(t, structs.CertificateMigration{Id: acmCertificateId(arnOther), Domain: "other.example.com", Result: resultError, Detail: "not used by this rack"}, got[acmCertificateId(arnOther)])
	assert.Equal(t, resultImported, got[acmCertificateId(arnApi)].Result)
	assert.Equal(t, structs.CertificateMigration{Id: "acm-000000000000", Result: resultError, Detail: "certificate not found"}, got["acm-000000000000"])
	assert.Empty(t, lookups(f.aws))
}

func TestCertificateMigrationInProgress(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, migrating: true, tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnRouter, domain: "*." + siteRouter, email: true, migrating: true})

	f.aws.readOnly = true

	ms, err := f.p.CertificateMigrationList(nil)
	require.NoError(t, err)
	got := migrations(ms)
	assert.Equal(t, resultWaiting, got[acmCertificateId(arnApex)].Result)
	assert.Equal(t, resultPublishing, got[acmCertificateId(arnRouter)].Result)

	dns, err := f.p.CertificateDns()
	require.NoError(t, err)
	require.Len(t, dns.Records, 1, "a migration started in the ACM console shows its record")
	assert.Equal(t, fakeRecord("example.com").Name, dns.Records[0].Name)
	assert.Empty(t, f.reg.requests, "read-only routes never call the registrar")

	f.aws.readOnly = false

	ms, err = f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	assert.Equal(t, resultPublishing, migrations(ms)[acmCertificateId(arnRouter)].Result)
	require.Len(t, f.reg.requests, 1)
	assert.Empty(t, lookups(f.aws), "a running migration needs no lookup")
	assert.Equal(t, 0, f.aws.count("UpdateCertificateOptions"))
}

func TestCertificatePendingDNS(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "new.example.com", status: "PENDING_VALIDATION", tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnWild, domain: "mail.example.com", status: "PENDING_VALIDATION", email: true, tags: map[string]string{"Rack": "convox"}})

	ms, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	got := migrations(ms)
	assert.Equal(t, resultWaiting, got[acmCertificateId(arnApex)].Result)
	assert.Equal(t, resultPending, got[acmCertificateId(arnWild)].Result)

	dns, err := f.p.CertificateDns()
	require.NoError(t, err)
	require.Len(t, dns.Records, 1, "a pending email certificate has no record")
	assert.Empty(t, dns.Preparing)

	f.publish("new.example.com")

	ms, err = f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	assert.Equal(t, resultPending, migrations(ms)[acmCertificateId(arnApex)].Result)
}

func TestCertificateRoute53(t *testing.T) {
	f := newCertificateFixture(t)

	f.p.Route53CertificateValidation = true
	f.aws.zones = []*fakeZone{
		{id: "ZSTALE", name: "example.com", ns: []string{"ns-9.awsdns-09.com"}, records: map[string]string{}},
		{id: "ZLIVE", name: "example.com", ns: []string{"ns-3.awsdns-03.org", "ns-4.awsdns-04.co.uk"}, records: map[string]string{}},
		{id: "ZPRIV", name: "example.com", private: true, ns: []string{"ns-3.awsdns-03.org", "ns-4.awsdns-04.co.uk"}, records: map[string]string{}},
	}

	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"Rack": "convox"}})

	ms, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	assert.Equal(t, resultPublishing, migrations(ms)[acmCertificateId(arnApex)].Result)

	rec := fakeRecord("example.com")
	assert.Equal(t, rec.Value, f.aws.zone("ZLIVE").records[dnsName(rec.Name)])
	assert.Empty(t, f.aws.zone("ZSTALE").records, "an undelegated zone is never written")

	ms, err = f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	assert.Equal(t, resultPublishing, migrations(ms)[acmCertificateId(arnApex)].Result, "an existing record with the same value counts as written")

	f.aws.zone("ZLIVE").records[dnsName(rec.Name)] = "_other.acm-validations.aws."

	ms, err = f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	assert.Equal(t, resultWaiting, migrations(ms)[acmCertificateId(arnApex)].Result, "a different value is left alone")
	assert.Equal(t, "_other.acm-validations.aws.", f.aws.zone("ZLIVE").records[dnsName(rec.Name)])

	f.p.Route53CertificateValidation = false

	dns, err := f.p.CertificateDns()
	require.NoError(t, err)
	require.Len(t, dns.Records, 1)
	assert.True(t, dns.Records[0].Route53)

	f.aws.errors["route53"] = "AccessDenied"

	dns, err = f.p.CertificateDns()
	require.NoError(t, err, "a Route 53 error never fails the request")
	require.Len(t, dns.Records, 1)
	assert.False(t, dns.Records[0].Route53)
}

func TestCertificateDnsReadOnly(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnWild, domain: "shop.example.com", sans: []string{"*.shop.example.com"}})
	f.add(&fakeCert{arn: arnRouter, domain: "*." + siteRouter, email: true})

	f.aws.readOnly = true

	dns, err := f.p.CertificateDns()
	require.NoError(t, err)

	assert.Equal(t, []string{acmCertificateId(arnApex)}, dns.Preparing)
	assert.Empty(t, dns.Errors)
	assert.Empty(t, dns.Records, "an unowned certificate's records are not listed")
	assert.Empty(t, f.reg.requests)
	assert.Empty(t, lookups(f.aws))

	f.aws.certs[1].tags["Rack"] = "convox"

	dns, err = f.p.CertificateDns()
	require.NoError(t, err)
	require.Len(t, dns.Records, 1)
	assert.Equal(t, structs.CertificateDnsRecord{
		Zone:         "example.com",
		Name:         fakeRecord("shop.example.com").Name,
		Type:         "CNAME",
		Value:        fakeRecord("shop.example.com").Value,
		Check:        checkMissing,
		Certificates: []string{acmCertificateId(arnWild)},
	}, dns.Records[0])
}

func TestCertificateDeadline(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnWild, domain: "shop.example.com", email: true, tags: map[string]string{"Rack": "convox"}})
	f.aws.s3[certificateCacheKey], _ = json.Marshal(certificateCache{Records: map[string]certificateRecord{"shop.example.com": fakeRecord("shop.example.com")}})

	saved := certificateAPIDeadline
	certificateAPIDeadline = 200 * time.Millisecond
	certificateDNSTimeout = 3 * time.Second
	t.Cleanup(func() { certificateAPIDeadline = saved })

	f.dns.hang = true

	start := time.Now()

	dns, err := f.p.CertificateDns()
	require.NoError(t, err)
	assert.Less(t, time.Since(start), 2*time.Second, "only the route deadline bounds the checks")
	require.Len(t, dns.Records, 2)
	assert.Equal(t, checkUnknown, dns.Records[0].Check)

	start = time.Now()

	ms, err := f.p.CertificateMigrate(nil)
	require.NoError(t, err)
	assert.Less(t, time.Since(start), 2*time.Second, "only the route deadline bounds the checks")
	assert.Equal(t, resultDNS, migrations(ms)[acmCertificateId(arnApex)].Result)
	assert.Equal(t, resultPreparing, migrations(ms)[acmCertificateId(arnWild)].Result)
}

func TestCertificatePanicInGoroutine(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnWild, domain: "dns.example.com", tags: map[string]string{"Rack": "convox"}})

	f.dns.panic = true

	ms, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	got := migrations(ms)
	assert.Equal(t, resultError, got[acmCertificateId(arnApex)].Result)
	assert.Contains(t, got[acmCertificateId(arnApex)].Detail, "panic")
	assert.Equal(t, resultDNS, got[acmCertificateId(arnWild)].Result)

	f.dns.panic = false

	assert.NotPanics(t, f.p.certificatesTick)
}

func TestCertificateRegistrarStatus(t *testing.T) {
	for _, c := range []struct {
		status  int
		message string
		held    bool
		detail  string
	}{
		{503, "registrar unavailable", false, "registrar 503: registrar unavailable"},
		{400, "load balancer not found", false, "registrar 400: load balancer not found"},
		{400, "load balancer not registered", false, "registrar 400: load balancer not registered"},
		{400, "invalid record name", true, "registrar 400: invalid record name"},
		{403, "load balancer belongs to another account", true, "registrar 403: load balancer belongs to another account"},
		{409, "record exists with a different value", true, "registrar 409: record exists with a different value, contact Convox support"},
		{429, "record limit reached", true, "registrar 429: record limit reached, contact Convox support"},
	} {
		t.Run(c.message, func(t *testing.T) {
			f := newCertificateFixture(t)

			f.add(&fakeCert{arn: arnRouter, domain: "*." + siteRouter, email: true})
			f.reg.status, f.reg.message = c.status, c.message

			ms, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
			require.NoError(t, err)

			m := migrations(ms)[acmCertificateId(arnRouter)]
			assert.Equal(t, resultError, m.Result)
			assert.Equal(t, c.detail, m.Detail)

			_, err = f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
			require.NoError(t, err)

			if c.held {
				assert.Equal(t, 1, f.reg.count(), "the worker holds the record for an hour")
			} else {
				assert.Equal(t, 2, f.reg.count(), "the worker retries next pass")
			}

			_, err = f.p.CertificateMigrate(nil)
			require.NoError(t, err)

			assert.Equal(t, map[bool]int{true: 2, false: 3}[c.held], f.reg.count(), "convox certs migrate always sends")
		})
	}
}

func TestCertificateVPCOnly(t *testing.T) {
	f := newCertificateFixture(t)

	f.dns.down["1.1.1.1"] = true
	f.dns.down["8.8.8.8"] = true
	f.dns.vpc = map[string]string{}

	f.add(&fakeCert{arn: arnRouter, domain: "*." + siteRouter, email: true, migrating: true})

	_, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	require.Equal(t, 1, f.reg.count(), "the record goes to the registrar before its first check")

	for _, call := range f.dns.calls {
		if strings.HasPrefix(call, "CNAME ") && strings.Contains(call, "_") {
			assert.True(t, strings.HasPrefix(call, "CNAME  "), "checks use only the VPC resolver: %s", call)
		}
	}
}

func TestCertificateGenerateDNS(t *testing.T) {
	f := newCertificateFixture(t)
	fastCertificateListWait(t, 5*time.Second)

	cert, err := f.p.CertificateGenerate([]string{"example.com", "www.example.com"}, structs.CertificateGenerateOptions{Validation: options.String("dns")})
	require.NoError(t, err)

	assert.Equal(t, statusPending, cert.Status)
	assert.True(t, cert.Owned)
	assert.Regexp(t, regexp.MustCompile(`^acm-[0-9]{12}$`), cert.Id)

	reqs := lookups(f.aws)
	require.Len(t, reqs, 1)
	assert.Equal(t, "DNS", reqs[0]["ValidationMethod"])
	assert.Equal(t, []interface{}{"www.example.com"}, reqs[0]["SubjectAlternativeNames"])
	assert.ElementsMatch(t, []interface{}{
		map[string]interface{}{"Key": "System", "Value": "convox"},
		map[string]interface{}{"Key": "Rack", "Value": "convox"},
	}, reqs[0]["Tags"])

	dns, err := f.p.CertificateDns()
	require.NoError(t, err)
	assert.Len(t, dns.Records, 2)
}

func TestCertificateGenerateEmailDefault(t *testing.T) {
	f := newCertificateFixture(t)
	fastCertificateListWait(t, 5*time.Second)

	for _, opts := range []structs.CertificateGenerateOptions{{}, {Validation: options.String("email")}} {
		cert, err := f.p.CertificateGenerate([]string{"example.com", "www.example.com"}, opts)
		require.NoError(t, err)
		assert.Equal(t, statusPending, cert.Status)
		assert.False(t, cert.Owned)
	}

	reqs := lookups(f.aws)
	require.Len(t, reqs, 2)

	for _, r := range reqs {
		assert.Nil(t, r["ValidationMethod"], "an older CLI gets the email request it got before")
		assert.Nil(t, r["Tags"])
		assert.Equal(t, []interface{}{"www.example.com"}, r["SubjectAlternativeNames"])
	}

	assert.Equal(t, 0, f.aws.count("DescribeCertificate"), "no record wait for an email certificate")

	_, err := f.p.CertificateGenerate([]string{"example.com"}, structs.CertificateGenerateOptions{Validation: options.String("http")})
	assert.EqualError(t, err, "validation must be dns or email")
	assert.Len(t, lookups(f.aws), 2)
}

func TestCertificateLogs(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"System": "convox", "Type": "app", "Rack": "convox", "Name": "web"}})

	_, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	require.Len(t, f.aws.logs, 1)
	assert.Contains(t, f.aws.logs[0], "certificate "+acmCertificateId(arnApex)+" (example.com) needs a DNS record before 2027-09-30: run convox certs dns")
	assert.Contains(t, f.aws.logs[0], "system/certificates")

	_, err = f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	assert.Len(t, f.aws.logs, 1, "once an hour")

	saved := certificateLogInterval
	certificateLogInterval = 0
	t.Cleanup(func() { certificateLogInterval = saved })

	_, err = f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	assert.Len(t, f.aws.logs, 2)

	_, err = f.p.CertificateMigrate(nil)
	require.NoError(t, err)
	assert.Len(t, f.aws.logs, 2, "only the worker logs")
}

func TestCertificateSteadyState(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", tags: map[string]string{"Rack": "convox"}})

	_, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	describes := f.aws.count("DescribeCertificate")

	_, err = f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	assert.Equal(t, describes, f.aws.count("DescribeCertificate"), "the worker skips certificates it saw on DNS")

	_, err = f.p.CertificateMigrate(nil)
	require.NoError(t, err)
	assert.Greater(t, f.aws.count("DescribeCertificate"), describes, "convox certs migrate reads every certificate")
}

func TestCertificateSwitchRecordChanged(t *testing.T) {
	f := newCertificateFixture(t)

	key := siteRouter
	stale := certificateRecord{Name: fakeRecord(key).Name, Value: "_stale.abcdefghij.acm-validations.aws."}

	f.aws.s3[certificateCacheKey], _ = json.Marshal(certificateCache{Records: map[string]certificateRecord{key: stale}})
	f.dns.set(stale.Name, stale.Value)

	f.add(&fakeCert{arn: arnRouter, domain: "*." + siteRouter, email: true})

	_, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	assert.Equal(t, 1, f.aws.count("UpdateCertificateOptions"))
	require.Equal(t, 1, f.reg.count(), "the requested record differs from the cached one and is published at once")
	assert.Equal(t, fakeRecord(key).Value, f.reg.requests[0].Value)

	var cache certificateCache
	require.NoError(t, json.Unmarshal(f.aws.s3[certificateCacheKey], &cache))
	assert.Equal(t, fakeRecord(key), cache.Records[key])
}

func TestCertificateCacheMerge(t *testing.T) {
	f := newCertificateFixture(t)

	f.aws.s3[certificateCacheKey] = []byte(`{"records":{"kept.example.com":{"name":"_a.kept.example.com.","value":"_b.x.acm-validations.aws."}}}`)

	r, err := f.p.certificateRun(f.p.Context(), time.Now().Add(time.Minute), true, true)
	require.NoError(t, err)

	f.aws.s3[certificateCacheKey] = []byte(`{"records":{"kept.example.com":{"name":"_a.kept.example.com.","value":"_b.x.acm-validations.aws."},"other.example.com":{"name":"_c.other.example.com.","value":"_d.x.acm-validations.aws."}}}`)

	r.learn(map[string]certificateRecord{"new.example.com": {Name: "_e.new.example.com.", Value: "_f.x.acm-validations.aws."}})

	var cache certificateCache
	require.NoError(t, json.Unmarshal(f.aws.s3[certificateCacheKey], &cache))
	assert.Len(t, cache.Records, 3, "the writer re-reads and merges")

	delete(f.aws.s3, certificateCacheKey)

	r2, err := f.p.certificateRun(f.p.Context(), time.Now().Add(time.Minute), true, true)
	require.NoError(t, err)

	_, ok := r2.record("new.example.com")
	assert.True(t, ok, "records learned in this process survive a lost cache object")
}

func TestCertificateGroups(t *testing.T) {
	custom := &acmCertificate{id: "acm-3", domain: "example.com"}
	svc := &acmCertificate{id: "acm-2", domain: "*." + siteRouter, sans: []string{"*.web-api." + siteRouter}}
	lb := &acmCertificate{id: "acm-1", domain: "*." + siteRouter}

	certs := []*acmCertificate{custom, svc, lb}

	sort.SliceStable(certs, func(i, j int) bool { return certificateOrder(certs[i]) < certificateOrder(certs[j]) })

	assert.Equal(t, [][]*acmCertificate{{lb}, {svc}, {custom}}, certificateGroups(certs))
}

func captureCertificateLog(t *testing.T) *[]string {
	lines := []string{}

	var mu sync.Mutex

	saved := certificateLogf
	certificateLogf = func(format string, args ...interface{}) {
		mu.Lock()
		defer mu.Unlock()
		lines = append(lines, fmt.Sprintf(format, args...))
	}
	t.Cleanup(func() { certificateLogf = saved })

	return &lines
}

func TestCertificateVPCOnlyHandPublished(t *testing.T) {
	f := newCertificateFixture(t)

	f.dns.down["1.1.1.1"] = true
	f.dns.down["8.8.8.8"] = true

	rec := fakeRecord(siteRouter)
	f.dns.vpc = map[string]string{dnsName(rec.Name): rec.Value}
	f.reg.status, f.reg.message = 503, "registrar unavailable"

	f.add(&fakeCert{arn: arnRouter, domain: "*." + siteRouter, email: true, migrating: true})
	f.add(&fakeCert{arn: arnSvc, domain: "*." + siteRouter, email: true, tags: map[string]string{"Rack": "convox"}})
	f.aws.s3[certificateCacheKey], _ = json.Marshal(certificateCache{Records: map[string]certificateRecord{siteRouter: rec}})

	ms, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	got := migrations(ms)
	assert.Equal(t, resultSwitching, got[acmCertificateId(arnSvc)].Result, "a record published by hand is seen even though the registrar fails")
	assert.Equal(t, 1, f.aws.count("UpdateCertificateOptions"))

	post, query := -1, -1
	for i, call := range f.dns.calls {
		if call == "POST "+dnsName(rec.Name) && post < 0 {
			post = i
		}
		if call == "CNAME  "+dnsName(rec.Name) && query < 0 {
			query = i
		}
	}
	require.GreaterOrEqual(t, post, 0)
	assert.Less(t, post, query, "the record goes to the registrar before the VPC resolver is asked for it")
}

func TestCertificateNoNameserverPublishFirst(t *testing.T) {
	f := newCertificateFixture(t)

	z := newDNSChecker(time.Time{}).zone(context.Background(), "convox.site")
	for _, ip := range z.servers {
		f.dns.down[ip] = true
	}

	f.add(&fakeCert{arn: arnRouter, domain: "*." + siteRouter, email: true, migrating: true})

	_, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	name := dnsName(fakeRecord(siteRouter).Name)
	post, query := -1, -1
	for i, call := range f.dns.calls {
		if call == "POST "+name && post < 0 {
			post = i
		}
		if call == "CNAME  "+name && query < 0 {
			query = i
		}
	}
	require.GreaterOrEqual(t, post, 0)
	assert.Less(t, post, query, "with no nameserver answering, the record is published before the VPC resolver sees the name")
}

func TestCertificateRegistrarLogLine(t *testing.T) {
	f := newCertificateFixture(t)
	lines := captureCertificateLog(t)

	f.add(&fakeCert{arn: arnRouter, domain: "*." + siteRouter, email: true, migrating: true})
	f.reg.status, f.reg.message = 403, "load balancer belongs to another account"

	_, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	rec := fakeRecord(siteRouter)
	assert.Contains(t, *lines, "certificate "+acmCertificateId(arnRouter)+" (*."+siteRouter+") record "+rec.Name+" CNAME "+rec.Value+": registrar 403: load balancer belongs to another account")
}

func TestCertificateDeadlineCut(t *testing.T) {
	f := newCertificateFixture(t)

	saved := certificateAPIDeadline
	certificateAPIDeadline = 300 * time.Millisecond
	certificateRecordWait = 5 * time.Second
	certificateLookupCutoff = 0
	t.Cleanup(func() { certificateAPIDeadline = saved; certificateLookupCutoff = 15 * time.Second })

	f.aws.lookupBare = true

	f.add(&fakeCert{arn: arnRouter, domain: "*." + siteRouter, email: true})
	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"Rack": "convox"}})

	rec := fakeRecord("example.com")
	f.aws.s3[certificateCacheKey], _ = json.Marshal(certificateCache{Records: map[string]certificateRecord{"example.com": rec}})
	f.publish("example.com")

	ms, err := f.p.CertificateMigrate(nil)
	require.NoError(t, err)

	got := migrations(ms)
	assert.Equal(t, resultSwitching, got[acmCertificateId(arnApex)].Result, "a later group is not starved by an earlier group's lookup")
	assert.Equal(t, resultPreparing, got[acmCertificateId(arnRouter)].Result, "work the deadline stopped is preparing, not error")
	assert.Equal(t, 1, f.aws.count("DeleteCertificate"), "the lookup is deleted after the deadline expired")
	assert.Len(t, f.aws.certs, 2)

	var cache certificateCache
	require.NoError(t, json.Unmarshal(f.aws.s3[certificateCacheKey], &cache))
	assert.Empty(t, cache.Errors, "a lookup the deadline stopped is not cached as an error")
}

func TestCertificateLookupBackoff(t *testing.T) {
	f := newCertificateFixture(t)

	f.aws.lookupFail = "ADDITIONAL_VERIFICATION_REQUIRED"
	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"Rack": "convox"}})

	ms, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	assert.Equal(t, "validation failed: ADDITIONAL_VERIFICATION_REQUIRED", migrations(ms)[acmCertificateId(arnApex)].Detail)
	assert.Equal(t, 1, f.aws.count("RequestCertificate"))
	assert.Equal(t, 1, f.aws.count("DeleteCertificate"))

	ms, err = f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	assert.Equal(t, resultError, migrations(ms)[acmCertificateId(arnApex)].Result)
	assert.Equal(t, 1, f.aws.count("RequestCertificate"), "the worker waits before another lookup")

	_, err = f.p.CertificateMigrate(nil)
	require.NoError(t, err)
	assert.Equal(t, 2, f.aws.count("RequestCertificate"), "convox certs migrate retries at once")
}

func TestCertificateReadErrorIsolated(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnRouter, domain: "*." + siteRouter, email: true, migrating: true})
	f.add(&fakeCert{arn: arnOther, domain: "other.example.org", email: true, lcdvError: "ValidationException"})
	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, lcdvError: "ValidationException", tags: map[string]string{"Rack": "convox"}})

	ms, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	got := migrations(ms)
	assert.Equal(t, resultPublishing, got[acmCertificateId(arnRouter)].Result)
	assert.Equal(t, resultError, got[acmCertificateId(arnApex)].Result)
	assert.Contains(t, got[acmCertificateId(arnApex)].Detail, "ValidationException")
	assert.NotContains(t, got, acmCertificateId(arnOther), "another tool's certificate failing does not stop the pass")

	_, err = f.p.CertificateDns()
	require.NoError(t, err)
}

func TestCertificateSwitchConflict(t *testing.T) {
	for code, want := range map[string]string{"ConflictException": resultSwitching, "LimitExceededException": resultError} {
		t.Run(code, func(t *testing.T) {
			f := newCertificateFixture(t)

			f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"Rack": "convox"}})
			f.aws.s3[certificateCacheKey], _ = json.Marshal(certificateCache{Records: map[string]certificateRecord{"example.com": fakeRecord("example.com")}})
			f.publish("example.com")
			f.aws.updateFail = code

			ms, err := f.p.CertificateMigrate(nil)
			require.NoError(t, err)

			m := migrations(ms)[acmCertificateId(arnApex)]
			assert.Equal(t, want, m.Result)
			if want == resultError {
				assert.Contains(t, m.Detail, code)
			}
		})
	}
}

func TestCertificateFailedResult(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", status: "FAILED", tags: map[string]string{"Rack": "convox"}})

	ms, err := f.p.CertificateMigrate([]string{acmCertificateId(arnApex)})
	require.NoError(t, err)
	assert.Equal(t, resultFailed, migrations(ms)[acmCertificateId(arnApex)].Result)
}

func TestCertificateIssuedLookupSwept(t *testing.T) {
	f := newCertificateFixture(t)

	lookup := f.add(&fakeCert{arn: arnApex, domain: "example.com", tags: map[string]string{"System": "convox", "Rack": "convox", "Purpose": lookupPurpose}, created: time.Now()})

	_, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	require.Len(t, f.aws.certs, 1, "too young to sweep")

	lookup.created = time.Now().Add(-time.Hour)

	_, err = f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)
	assert.Empty(t, f.aws.certs, "an issued lookup is never remembered as a DNS certificate, so the sweep still finds it")
}

func TestCertificateLookupDeleteChecks(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", status: "PENDING_VALIDATION", tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnWild, domain: "used.example.com", tags: map[string]string{"Rack": "convox", "Purpose": lookupPurpose}, inUse: []string{routerLB}})

	f.p.lookupDelete(arnApex)
	f.p.lookupDelete(arnWild)

	assert.Equal(t, 0, f.aws.count("DeleteCertificate"), "a certificate without the lookup tag, or in use, is never deleted")
}

func TestCertificateCacheWrittenBeforeDelete(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"Rack": "convox"}})

	_, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	require.Len(t, f.aws.deleted, 1)

	var cache certificateCache
	require.NoError(t, json.Unmarshal(f.aws.deleted[0], &cache))
	assert.Equal(t, fakeRecord("example.com"), cache.Records["example.com"])
}

func TestCertificateFailedCacheWrite(t *testing.T) {
	f := newCertificateFixture(t)

	f.aws.errors["S3PUT"] = "fail"
	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"Rack": "convox"}})

	_, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	_, err = f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	assert.Equal(t, 1, f.aws.count("RequestCertificate"), "records learned in this process survive a failed cache write")
}

func TestCertificateAuthCutoff(t *testing.T) {
	f := newCertificateFixture(t)

	saved := certificateAuthCutoff
	certificateAuthCutoff = 0
	t.Cleanup(func() { certificateAuthCutoff = saved })

	f.add(&fakeCert{arn: arnApex, domain: "example.com", tags: map[string]string{"Rack": "convox"}})

	_, err := f.p.CertificateDns()
	require.NoError(t, err)

	name := dnsName(fakeRecord("example.com").Name)
	for _, call := range f.dns.calls {
		if strings.HasPrefix(call, "CNAME ") && strings.HasSuffix(call, " "+name) {
			assert.Equal(t, "CNAME  "+name, call, "past the cutoff only the VPC resolver is asked")
		}
	}
}

func shortDeadline(t *testing.T, d time.Duration) {
	saved, cutoff := certificateAPIDeadline, certificateLookupCutoff
	certificateAPIDeadline = d
	certificateLookupCutoff = 0
	certificateDNSTimeout = 3 * time.Second
	t.Cleanup(func() { certificateAPIDeadline, certificateLookupCutoff = saved, cutoff })
}

func TestCertificateLookupFailureNotMasked(t *testing.T) {
	f := newCertificateFixture(t)
	shortDeadline(t, 400*time.Millisecond)
	certificateRecordWait = 5 * time.Second

	f.aws.lookupBare = true
	f.aws.lookupFor = map[string]string{"bad.example.com": "DOMAIN_NOT_ALLOWED"}

	f.add(&fakeCert{arn: arnApex, domain: "bad.example.com", email: true, tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnWild, domain: "slow.example.com", email: true, tags: map[string]string{"Rack": "convox"}})

	ms, err := f.p.CertificateMigrate(nil)
	require.NoError(t, err)

	got := migrations(ms)
	assert.Equal(t, structs.CertificateMigration{Id: acmCertificateId(arnApex), Domain: "bad.example.com", Result: resultError, Detail: "validation failed: DOMAIN_NOT_ALLOWED"}, got[acmCertificateId(arnApex)], "a real failure is shown even when the deadline stopped a sibling")
	assert.Equal(t, resultPreparing, got[acmCertificateId(arnWild)].Result)
}

func TestCertificateCutPerResult(t *testing.T) {
	f := newCertificateFixture(t)
	shortDeadline(t, 300*time.Millisecond)

	f.dns.zone("example.org", "ns-5.awsdns-05.com")
	z := newDNSChecker(time.Time{}).zone(context.Background(), "example.org")
	for _, ip := range z.servers {
		f.dns.down[ip] = true
	}

	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnWild, domain: "shop.example.org", email: true, tags: map[string]string{"Rack": "convox"}})
	f.aws.s3[certificateCacheKey], _ = json.Marshal(certificateCache{Records: map[string]certificateRecord{
		"example.com":      fakeRecord("example.com"),
		"shop.example.org": fakeRecord("shop.example.org"),
	}})

	ms, err := f.p.CertificateMigrate(nil)
	require.NoError(t, err)

	got := migrations(ms)
	assert.Equal(t, resultWaiting, got[acmCertificateId(arnApex)].Result, "a result finished before the deadline keeps its token")
	assert.Equal(t, resultPreparing, got[acmCertificateId(arnWild)].Result)
}

func TestCertificateMigrationListCachedError(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"Rack": "convox"}})
	f.aws.s3[certificateCacheKey], _ = json.Marshal(certificateCache{Errors: map[string]certificateCacheError{acmCertificateId(arnApex): {Error: "validation failed: CAA_ERROR", Time: time.Now()}}})
	f.aws.readOnly = true

	ms, err := f.p.CertificateMigrationList([]string{acmCertificateId(arnApex)})
	require.NoError(t, err)
	assert.Equal(t, structs.CertificateMigration{Id: acmCertificateId(arnApex), Domain: "example.com", Result: resultError, Detail: "validation failed: CAA_ERROR"}, migrations(ms)[acmCertificateId(arnApex)])
}

func TestCertificateMigrateNeverSwitchesOnVPCAnswer(t *testing.T) {
	f := newCertificateFixture(t)

	saved := certificateAuthCutoff
	certificateAuthCutoff = 0
	t.Cleanup(func() { certificateAuthCutoff = saved })

	rec := fakeRecord("example.com")
	f.dns.vpc = map[string]string{dnsName(rec.Name): rec.Value}

	f.add(&fakeCert{arn: arnApex, domain: "example.com", email: true, tags: map[string]string{"Rack": "convox"}})
	f.aws.s3[certificateCacheKey], _ = json.Marshal(certificateCache{Records: map[string]certificateRecord{"example.com": rec}})

	ms, err := f.p.CertificateMigrate(nil)
	require.NoError(t, err)
	assert.Equal(t, resultWaiting, migrations(ms)[acmCertificateId(arnApex)].Result, "a record only a private zone answers never starts a switch")
	assert.Equal(t, 0, f.aws.count("UpdateCertificateOptions"))
}

func TestCertificateMigrationListSiteVPC(t *testing.T) {
	f := newCertificateFixture(t)

	f.dns.down["1.1.1.1"] = true
	f.dns.down["8.8.8.8"] = true
	f.dns.vpc = map[string]string{}

	f.add(&fakeCert{arn: arnRouter, domain: "*." + siteRouter, email: true})
	f.aws.s3[certificateCacheKey], _ = json.Marshal(certificateCache{Records: map[string]certificateRecord{siteRouter: fakeRecord(siteRouter)}})

	ms, err := f.p.CertificateMigrationList(nil)
	require.NoError(t, err)
	assert.Equal(t, resultPublishing, migrations(ms)[acmCertificateId(arnRouter)].Result)
	assert.Equal(t, 0, f.reg.count())

	for _, call := range f.dns.calls {
		assert.NotEqual(t, "CNAME  "+dnsName(fakeRecord(siteRouter).Name), call, "a read-only run never asks the VPC resolver about an unpublished convox.site name")
	}
}

func TestCertificateRegistrarLogPerRecord(t *testing.T) {
	f := newCertificateFixture(t)
	lines := captureCertificateLog(t)

	f.add(&fakeCert{arn: arnSvc, domain: "*." + siteRouter, sans: []string{"*.web-api." + siteRouter}, email: true, migrating: true, tags: map[string]string{"Rack": "convox"}})
	f.reg.status, f.reg.message = 503, "registrar unavailable"

	_, err := f.p.certificatePass(nil, time.Now().Add(time.Minute), true)
	require.NoError(t, err)

	joined := strings.Join(*lines, "\n")
	assert.Contains(t, joined, fakeRecord(siteRouter).Name)
	assert.Contains(t, joined, fakeRecord("web-api."+siteRouter).Name)
}

func TestCertificateDnsLargeAccount(t *testing.T) {
	f := newCertificateFixture(t)
	shortDeadline(t, 350*time.Millisecond)
	acmPaceInterval = 100 * time.Millisecond

	for i := 0; i < 20; i++ {
		f.add(&fakeCert{arn: fmt.Sprintf("arn:aws:acm:us-test-1:123456789012:certificate/77777777-0000-0000-0000-%012d", i), domain: fmt.Sprintf("other%d.example.net", i)})
	}
	f.add(&fakeCert{arn: arnApex, domain: "example.com", tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnGen1, domain: "d.example.com"})

	dns, err := f.p.CertificateDns()
	require.NoError(t, err)
	assert.Contains(t, dns.Preparing, acmCertificateId(arnApex), "a certificate the deadline kept from being read is reported, not dropped")
	require.Len(t, dns.Records, 1, "the gen1 certificate is read first although ACM lists it last")
	assert.Equal(t, []string{acmCertificateId(arnGen1)}, dns.Records[0].Certificates)

	time.Sleep(600 * time.Millisecond)

	dns, err = f.p.CertificateDns()
	require.NoError(t, err)
	assert.Len(t, dns.Records, 1, "reads the deadline cancelled do not use up later calls' pace")

	time.Sleep(600 * time.Millisecond)

	ms, err := f.p.CertificateMigrate(nil)
	require.NoError(t, err)
	assert.Equal(t, resultDNS, migrations(ms)[acmCertificateId(arnGen1)].Result, "convox certs migrate reads likely certificates first too")
}

func TestCertificateDnsOwnedReadError(t *testing.T) {
	f := newCertificateFixture(t)

	f.add(&fakeCert{arn: arnApex, domain: "example.com", describe: "AccessDeniedException", tags: map[string]string{"Rack": "convox"}})
	f.add(&fakeCert{arn: arnOther, domain: "other.example.net", describe: "AccessDeniedException"})

	dns, err := f.p.CertificateDns()
	require.NoError(t, err)
	require.Len(t, dns.Errors, 1)
	assert.Equal(t, acmCertificateId(arnApex), dns.Errors[0].Id)
	assert.Contains(t, dns.Errors[0].Error, "AccessDeniedException")
	assert.Empty(t, dns.Preparing)
}
