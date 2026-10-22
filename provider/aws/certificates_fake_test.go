package aws

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeCert struct {
	arn       string
	created   time.Time
	describe  string
	domain    string
	email     bool
	failure   string
	lcdvError string
	inUse     []string
	kind      string
	migrating bool
	noRecords bool
	notAfter  time.Time
	sans      []string
	status    string
	switched  bool
	tags      map[string]string
}

type fakeResource struct {
	id   string
	kind string
	name string
}

type fakeStack struct {
	name   string
	params map[string]string
	tags   map[string]string
}

type fakeZone struct {
	id      string
	name    string
	ns      []string
	private bool
	records map[string]string
}

// fakeAWS serves the ACM, CloudFormation, S3, Route 53 and CloudWatch Logs calls the certificate
// code makes, with the real wire formats, and keeps their state.
type fakeAWS struct {
	t *testing.T

	mu         sync.Mutex
	certs      []*fakeCert
	calls      []string
	errors     map[string]string
	lcdvError  string
	deleted    [][]byte
	lookupBare bool
	lookupFail string
	lookupFor  map[string]string
	logs       []string
	readOnly   bool
	requests   []map[string]interface{}
	resources  []fakeResource
	s3         map[string][]byte
	stacks     []fakeStack
	zones      []*fakeZone
	updateFail string
}

func newFakeAWS(t *testing.T) *fakeAWS {
	return &fakeAWS{t: t, errors: map[string]string{}, s3: map[string][]byte{}}
}

func fakeRecord(key string) certificateRecord {
	sum := md5.Sum([]byte(key))
	val := md5.Sum([]byte("value:" + key))

	return certificateRecord{
		Name:  "_" + hex.EncodeToString(sum[:]) + "." + key + ".",
		Value: "_" + hex.EncodeToString(val[:]) + ".abcdefghij.acm-validations.aws.",
	}
}

func (f *fakeAWS) cert(arn string) *fakeCert {
	for _, c := range f.certs {
		if c.arn == arn {
			return c
		}
	}

	return nil
}

func (f *fakeAWS) count(op string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	n := 0

	for _, c := range f.calls {
		if c == op {
			n++
		}
	}

	return n
}

func (f *fakeAWS) provider() *Provider {
	p := stubProviderHandler(f.t, f)
	resetCertificateState(f.t)
	return p
}

func resetCertificateState(t *testing.T) {
	reset := func() {
		certificateState.Lock()
		certificateState.dns = map[string]bool{}
		certificateState.holds = map[string]registrarHold{}
		certificateState.logged = map[string]time.Time{}
		certificateState.records = map[string]certificateRecord{}
		certificateState.Unlock()

		acmPaceLock.Lock()
		acmPace = map[string]time.Time{}
		acmPaceLock.Unlock()
	}

	reset()
	t.Cleanup(reset)
}

func (f *fakeAWS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	f.mu.Lock()
	defer f.mu.Unlock()

	if target := r.Header.Get("X-Amz-Target"); target != "" {
		op := target[strings.Index(target, ".")+1:]
		f.calls = append(f.calls, op)

		var in map[string]interface{}
		_ = json.Unmarshal(body, &in)
		f.requests = append(f.requests, map[string]interface{}{"op": op, "body": in, "raw": string(body)})

		if code, ok := f.errors[op]; ok {
			w.WriteHeader(400)
			_, _ = fmt.Fprintf(w, `{"__type":%q,"message":"fake %s"}`, code, code)
			return
		}

		if strings.HasPrefix(target, "Logs_") {
			f.logs = append(f.logs, string(body))
			_, _ = fmt.Fprint(w, `{"nextSequenceToken":"1"}`)
			return
		}

		f.acm(w, op, in)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/2013-04-01/") {
		f.route53(w, r, body)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/convox-settings/") {
		key := strings.TrimPrefix(r.URL.Path, "/convox-settings/")
		f.calls = append(f.calls, "S3"+r.Method)

		switch r.Method {
		case http.MethodHead, http.MethodGet:
			data, ok := f.s3[key]
			if !ok {
				w.WriteHeader(404)
				return
			}
			if r.Method == http.MethodGet {
				_, _ = w.Write(data)
			}
		case http.MethodPut:
			if f.readOnly {
				f.t.Errorf("S3 write in a read-only run")
			}
			if _, ok := f.errors["S3PUT"]; ok {
				w.WriteHeader(403)
				_, _ = fmt.Fprint(w, `<Error><Code>AccessDenied</Code><Message>fake</Message></Error>`)
				return
			}
			f.s3[key] = body
		}
		return
	}

	form, _ := url.ParseQuery(string(body))
	f.calls = append(f.calls, form.Get("Action"))

	switch form.Get("Action") {
	case "ListServerCertificates":
		_, _ = fmt.Fprint(w, `<ListServerCertificatesResponse><ListServerCertificatesResult><ServerCertificateMetadataList/><IsTruncated>false</IsTruncated></ListServerCertificatesResult></ListServerCertificatesResponse>`)
	case "ListStackResources":
		f.listStackResources(w, form.Get("StackName"))
	case "DescribeStacks":
		f.describeStacks(w, form.Get("StackName"))
	default:
		f.t.Errorf("unexpected request %s %s %s", r.Method, r.URL.Path, body)
		w.WriteHeader(404)
	}
}

func epoch(t time.Time) string {
	if t.IsZero() {
		return "null"
	}
	return fmt.Sprintf("%d", t.Unix())
}

func (f *fakeAWS) acm(w http.ResponseWriter, op string, in map[string]interface{}) {
	arn, _ := in["CertificateArn"].(string)
	c := f.cert(arn)

	writes := map[string]bool{"RequestCertificate": true, "DeleteCertificate": true, "UpdateCertificateOptions": true}
	if f.readOnly && writes[op] {
		f.t.Errorf("ACM %s in a read-only run", op)
	}

	notFound := func() {
		w.WriteHeader(400)
		_, _ = fmt.Fprint(w, `{"__type":"ResourceNotFoundException","message":"not found"}`)
	}

	switch op {
	case "ListCertificates":
		items := []string{}
		for _, c := range f.certs {
			items = append(items, fmt.Sprintf(`{"CertificateArn":%q,"DomainName":%q,"Status":%q,"Type":%q,"InUse":%t,"CreatedAt":%s}`, c.arn, c.domain, c.status, c.kind, len(c.inUse) > 0, epoch(c.created)))
		}
		_, _ = fmt.Fprintf(w, `{"CertificateSummaryList":[%s]}`, strings.Join(items, ","))
	case "ListTagsForCertificate":
		if c == nil {
			notFound()
			return
		}
		tags := []string{}
		for k, v := range c.tags {
			tags = append(tags, fmt.Sprintf(`{"Key":%q,"Value":%q}`, k, v))
		}
		sort.Strings(tags)
		_, _ = fmt.Fprintf(w, `{"Tags":[%s]}`, strings.Join(tags, ","))
	case "DescribeCertificate":
		if c == nil {
			notFound()
			return
		}
		if c.describe != "" {
			w.WriteHeader(400)
			_, _ = fmt.Fprintf(w, `{"__type":%q,"message":"fake"}`, c.describe)
			return
		}
		dvos := []string{}
		for _, d := range append([]string{c.domain}, c.sans...) {
			method := "DNS"
			if c.email {
				method = "EMAIL"
			}
			rr := ""
			if method == "DNS" && !c.noRecords {
				rec := fakeRecord(certificateKey(d))
				rr = fmt.Sprintf(`,"ResourceRecord":{"Name":%q,"Type":"CNAME","Value":%q}`, rec.Name, rec.Value)
			}
			dvos = append(dvos, fmt.Sprintf(`{"DomainName":%q,"ValidationMethod":%q%s}`, d, method, rr))
		}
		sans, _ := json.Marshal(append([]string{c.domain}, c.sans...))
		inUse, _ := json.Marshal(c.inUse)
		_, _ = fmt.Fprintf(w, `{"Certificate":{"CertificateArn":%q,"DomainName":%q,"SubjectAlternativeNames":%s,"Status":%q,"Type":%q,"InUseBy":%s,"NotAfter":%s,"CreatedAt":%s,"FailureReason":%q,"DomainValidationOptions":[%s]}}`,
			c.arn, c.domain, sans, c.status, c.kind, inUse, epoch(c.notAfter), epoch(c.created), c.failure, strings.Join(dvos, ","))
	case "ListCertificateDomainValidations":
		if c == nil {
			notFound()
			return
		}
		if code := f.lcdvError + c.lcdvError; code != "" {
			w.WriteHeader(400)
			_, _ = fmt.Fprintf(w, `{"__type":%q,"message":"fake"}`, code)
			return
		}
		items := []string{}
		for _, d := range append([]string{c.domain}, c.sans...) {
			active := `{"ValidationMethod":"DNS","ValidationStatus":"SUCCESS"}`
			if c.email && !c.switched {
				active = `{"ValidationMethod":"EMAIL","ValidationStatus":"SUCCESS"}`
			}
			requested := ""
			if c.migrating {
				rec := fakeRecord(certificateKey(d))
				requested = fmt.Sprintf(`,"RequestedValidationConfiguration":{"ValidationMethod":"DNS","ValidationStatus":"PENDING_VALIDATION","ValidationChallenge":{"DnsValidationChallenge":{"ResourceRecord":{"Name":%q,"Type":"CNAME","Value":%q}}}}`, rec.Name, rec.Value)
			}
			items = append(items, fmt.Sprintf(`{"DomainName":%q,"ActiveValidationConfiguration":%s%s}`, d, active, requested))
		}
		_, _ = fmt.Fprintf(w, `{"DomainValidationSummaryList":[%s]}`, strings.Join(items, ","))
	case "UpdateCertificateOptions":
		if c == nil {
			notFound()
			return
		}
		if f.updateFail != "" {
			w.WriteHeader(400)
			_, _ = fmt.Fprintf(w, `{"__type":%q,"message":"fake"}`, f.updateFail)
			return
		}
		c.migrating = true
		_, _ = fmt.Fprint(w, `{}`)
	case "RequestCertificate":
		domain, _ := in["DomainName"].(string)
		sans := []string{}
		if ss, ok := in["SubjectAlternativeNames"].([]interface{}); ok {
			for _, s := range ss {
				sans = append(sans, s.(string))
			}
		}
		tags := map[string]string{}
		if ts, ok := in["Tags"].([]interface{}); ok {
			for _, t := range ts {
				m := t.(map[string]interface{})
				tags[m["Key"].(string)] = m["Value"].(string)
			}
		}
		n := len(f.certs) + 1
		arn := fmt.Sprintf("arn:aws:acm:us-test-1:123456789012:certificate/00000000-0000-0000-0000-%012d", n)
		status, failure := "PENDING_VALIDATION", f.lookupFail
		if reason, ok := f.lookupFor[domain]; ok {
			failure = reason
		}
		if failure != "" {
			status = "FAILED"
		}
		f.certs = append(f.certs, &fakeCert{arn: arn, domain: domain, sans: sans, kind: "AMAZON_ISSUED", noRecords: f.lookupBare || failure != "", failure: failure, status: status, tags: tags, created: time.Now()})
		_, _ = fmt.Fprintf(w, `{"CertificateArn":%q}`, arn)
	case "DeleteCertificate":
		if c == nil {
			notFound()
			return
		}
		f.deleted = append(f.deleted, append([]byte{}, f.s3[certificateCacheKey]...))
		for i := range f.certs {
			if f.certs[i].arn == arn {
				f.certs = append(f.certs[:i], f.certs[i+1:]...)
				break
			}
		}
		_, _ = fmt.Fprint(w, `{}`)
	default:
		f.t.Errorf("unexpected ACM operation %s", op)
		w.WriteHeader(400)
	}
}

func (f *fakeAWS) listStackResources(w http.ResponseWriter, stack string) {
	members := []string{}

	for _, r := range f.resources {
		members = append(members, fmt.Sprintf(`<member><LogicalResourceId>%s</LogicalResourceId><PhysicalResourceId>%s</PhysicalResourceId><ResourceType>%s</ResourceType><ResourceStatus>CREATE_COMPLETE</ResourceStatus><LastUpdatedTimestamp>2026-01-01T00:00:00Z</LastUpdatedTimestamp></member>`, r.name, r.id, r.kind))
	}

	if stack != "convox" {
		members = []string{`<member><LogicalResourceId>LogGroup</LogicalResourceId><PhysicalResourceId>` + stack + `-LogGroup</PhysicalResourceId><ResourceType>AWS::Logs::LogGroup</ResourceType><ResourceStatus>CREATE_COMPLETE</ResourceStatus><LastUpdatedTimestamp>2026-01-01T00:00:00Z</LastUpdatedTimestamp></member>`}
	}

	_, _ = fmt.Fprintf(w, `<ListStackResourcesResponse><ListStackResourcesResult><StackResourceSummaries>%s</StackResourceSummaries></ListStackResourcesResult></ListStackResourcesResponse>`, strings.Join(members, ""))
}

func (f *fakeAWS) describeStacks(w http.ResponseWriter, name string) {
	members := []string{}

	for _, s := range f.stacks {
		if name != "" && s.name != name {
			continue
		}

		params, tags := "", ""
		for k, v := range s.params {
			params += fmt.Sprintf(`<member><ParameterKey>%s</ParameterKey><ParameterValue>%s</ParameterValue></member>`, k, v)
		}
		for k, v := range s.tags {
			tags += fmt.Sprintf(`<member><Key>%s</Key><Value>%s</Value></member>`, k, v)
		}

		members = append(members, fmt.Sprintf(`<member><StackName>%s</StackName><StackId>%s</StackId><StackStatus>UPDATE_COMPLETE</StackStatus><CreationTime>2026-01-01T00:00:00Z</CreationTime><Parameters>%s</Parameters><Tags>%s</Tags></member>`, s.name, s.name, params, tags))
	}

	_, _ = fmt.Fprintf(w, `<DescribeStacksResponse><DescribeStacksResult><Stacks>%s</Stacks></DescribeStacksResult></DescribeStacksResponse>`, strings.Join(members, ""))
}

func (f *fakeAWS) zone(id string) *fakeZone {
	for _, z := range f.zones {
		if z.id == id {
			return z
		}
	}

	return nil
}

func (f *fakeAWS) route53(w http.ResponseWriter, r *http.Request, body []byte) {
	path := strings.TrimPrefix(r.URL.Path, "/2013-04-01/")

	if code, ok := f.errors["route53"]; ok {
		f.calls = append(f.calls, "Route53Error")
		w.WriteHeader(403)
		_, _ = fmt.Fprintf(w, `<ErrorResponse><Error><Type>Sender</Type><Code>%s</Code><Message>fake</Message></Error></ErrorResponse>`, code)
		return
	}

	switch {
	case path == "hostedzonesbyname":
		f.calls = append(f.calls, "ListHostedZonesByName")
		name := dnsName(r.URL.Query().Get("dnsname"))
		zones := ""
		for _, z := range f.zones {
			if z.name >= name {
				zones += fmt.Sprintf(`<HostedZone><Id>/hostedzone/%s</Id><Name>%s.</Name><CallerReference>ref</CallerReference><Config><PrivateZone>%t</PrivateZone></Config></HostedZone>`, z.id, z.name, z.private)
			}
		}
		_, _ = fmt.Fprintf(w, `<ListHostedZonesByNameResponse><HostedZones>%s</HostedZones><DNSName>%s</DNSName><IsTruncated>false</IsTruncated><MaxItems>100</MaxItems></ListHostedZonesByNameResponse>`, zones, name)
	case strings.HasPrefix(path, "hostedzone/") && strings.HasSuffix(path, "/rrset") && r.Method == http.MethodGet:
		f.calls = append(f.calls, "ListResourceRecordSets")
		z := f.zone(strings.Split(path, "/")[1])
		name := dnsName(r.URL.Query().Get("name"))
		sets := ""
		if v, ok := z.records[name]; ok {
			sets = fmt.Sprintf(`<ResourceRecordSet><Name>%s.</Name><Type>CNAME</Type><TTL>300</TTL><ResourceRecords><ResourceRecord><Value>%s</Value></ResourceRecord></ResourceRecords></ResourceRecordSet>`, name, v)
		}
		_, _ = fmt.Fprintf(w, `<ListResourceRecordSetsResponse><ResourceRecordSets>%s</ResourceRecordSets><IsTruncated>false</IsTruncated><MaxItems>1</MaxItems></ListResourceRecordSetsResponse>`, sets)
	case strings.HasPrefix(path, "hostedzone/") && strings.HasSuffix(path, "/rrset/"):
		f.calls = append(f.calls, "ChangeResourceRecordSets")
		if f.readOnly {
			f.t.Errorf("Route 53 write in a read-only run")
		}
		z := f.zone(strings.Split(path, "/")[1])
		s := string(body)
		name := dnsName(between(s, "<Name>", "</Name>"))
		if _, ok := z.records[name]; ok {
			w.WriteHeader(400)
			_, _ = fmt.Fprint(w, `<InvalidChangeBatch><Messages><Message>Tried to create resource record set but it already exists</Message></Messages></InvalidChangeBatch>`)
			return
		}
		if !strings.Contains(s, "<Action>CREATE</Action>") || !strings.Contains(s, "<TTL>300</TTL>") {
			f.t.Errorf("unexpected change batch %s", s)
		}
		z.records[name] = between(s, "<Value>", "</Value>")
		_, _ = fmt.Fprint(w, `<ChangeResourceRecordSetsResponse><ChangeInfo><Id>/change/C1</Id><Status>PENDING</Status><SubmittedAt>2026-01-01T00:00:00Z</SubmittedAt></ChangeInfo></ChangeResourceRecordSetsResponse>`)
	case strings.HasPrefix(path, "hostedzone/"):
		f.calls = append(f.calls, "GetHostedZone")
		z := f.zone(strings.TrimPrefix(path, "hostedzone/"))
		ns := ""
		for _, n := range z.ns {
			ns += "<NameServer>" + n + "</NameServer>"
		}
		_, _ = fmt.Fprintf(w, `<GetHostedZoneResponse><HostedZone><Id>/hostedzone/%s</Id><Name>%s.</Name><CallerReference>ref</CallerReference></HostedZone><DelegationSet><NameServers>%s</NameServers></DelegationSet></GetHostedZoneResponse>`, z.id, z.name, ns)
	default:
		f.t.Errorf("unexpected Route 53 request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(404)
	}
}

func between(s, a, b string) string {
	i := strings.Index(s, a)
	if i < 0 {
		return ""
	}

	s = s[i+len(a):]

	if j := strings.Index(s, b); j >= 0 {
		return s[:j]
	}

	return s
}

// fakeDNS answers like public DNS for a set of zones: public resolvers give delegations, the
// zones' nameservers answer authoritatively, and the VPC resolver sees what vpc says, or the
// public answer when vpc is nil.
type fakeDNS struct {
	mu        sync.Mutex
	aliases   map[string]string
	calls     []string
	down      map[string]bool
	hang      bool
	panic     bool
	records   map[string]string
	refused   map[string]bool
	vpc       map[string]string
	zones     map[string][]string
	nsServers map[string]string
}

func newFakeDNS() *fakeDNS {
	return &fakeDNS{
		aliases:   map[string]string{},
		down:      map[string]bool{},
		records:   map[string]string{},
		refused:   map[string]bool{},
		zones:     map[string][]string{},
		nsServers: map[string]string{},
	}
}

func (d *fakeDNS) zone(name string, hosts ...string) {
	d.zones[name] = hosts
	for i, h := range hosts {
		d.nsServers[h] = fmt.Sprintf("192.0.2.%d", len(d.nsServers)+i+1)
	}
}

func (d *fakeDNS) install(t *testing.T) {
	old := certificateDNS
	certificateDNS = d
	t.Cleanup(func() { certificateDNS = old })
}

func (d *fakeDNS) set(name, value string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.records[dnsName(name)] = value
}

func notFoundErr(name string) error {
	return &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
}

func (d *fakeDNS) enter(ctx context.Context, server, kind, name string) error {
	d.mu.Lock()
	d.calls = append(d.calls, kind+" "+server+" "+dnsName(name))
	hang, down, refused, p := d.hang, d.down[server], d.refused[server], d.panic
	d.mu.Unlock()

	if p {
		panic("resolver panic")
	}

	if hang || down {
		<-ctx.Done()
		return &net.DNSError{Err: "i/o timeout", Name: name, IsTimeout: true}
	}

	if refused {
		return &net.DNSError{Err: "server misbehaving", Name: name}
	}

	return nil
}

func (d *fakeDNS) public(server string) bool {
	for _, s := range certificatePublicResolvers {
		if s == server {
			return true
		}
	}
	return false
}

func (d *fakeDNS) LookupNS(ctx context.Context, server, name string) ([]*net.NS, error) {
	if err := d.enter(ctx, server, "NS", name); err != nil {
		return nil, err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	n := dnsName(name)

	if target, ok := d.aliases[n]; ok {
		n = target
	}

	hosts, ok := d.zones[n]
	if !ok {
		return nil, notFoundErr(name)
	}

	ns := []*net.NS{}
	for _, h := range hosts {
		ns = append(ns, &net.NS{Host: h + "."})
	}

	return ns, nil
}

func (d *fakeDNS) LookupHost(ctx context.Context, server, name string) ([]string, error) {
	if err := d.enter(ctx, server, "A", name); err != nil {
		return nil, err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if ip, ok := d.nsServers[dnsName(name)]; ok {
		return []string{ip}, nil
	}

	return nil, notFoundErr(name)
}

func (d *fakeDNS) LookupCNAME(ctx context.Context, server, name string) (string, error) {
	if err := d.enter(ctx, server, "CNAME", name); err != nil {
		return "", err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	n := dnsName(name)

	if d.public(server) {
		if target, ok := d.aliases[n]; ok {
			return target + ".", nil
		}
		return "", notFoundErr(name)
	}

	if server == "" && d.vpc != nil {
		if v, ok := d.vpc[n]; ok {
			return v, nil
		}
		return "", notFoundErr(name)
	}

	if v, ok := d.records[n]; ok {
		return v, nil
	}

	for i := 0; i < strings.Count(n, "."); i++ {
		if v, ok := d.records["*."+strings.SplitN(n, ".", i+2)[i+1]]; ok {
			return v, nil
		}
	}

	return "", notFoundErr(name)
}

type fakeRegistrar struct {
	mu       sync.Mutex
	dns      *fakeDNS
	requests []registrarRequest
	status   int
	message  string
}

func (r *fakeRegistrar) install(t *testing.T) {
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	old := certificateRegistrarURL
	certificateRegistrarURL = srv.URL + "/validation"
	t.Cleanup(func() { certificateRegistrarURL = old })
}

func (r *fakeRegistrar) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var in registrarRequest
	_ = json.NewDecoder(req.Body).Decode(&in)

	r.mu.Lock()
	r.requests = append(r.requests, in)
	status, message := r.status, r.message
	r.mu.Unlock()

	if r.dns != nil {
		r.dns.mu.Lock()
		r.dns.calls = append(r.dns.calls, "POST "+dnsName(in.Name))
		r.dns.mu.Unlock()
	}

	if status == 0 {
		status = 200
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if status != 200 {
		_, _ = fmt.Fprintf(w, `{"error":%q}`, message)
		return
	}

	if r.dns != nil {
		r.dns.set(in.Name, in.Value)
	}

	_, _ = fmt.Fprintf(w, `{"name":%q,"value":%q}`, in.Name, in.Value)
}

func (r *fakeRegistrar) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.requests)
}
