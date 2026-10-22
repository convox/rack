package aws

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/cloudformation"
)

type acmCertificate struct {
	arn        string
	created    time.Time
	domain     string
	email      bool
	err        error
	expiration time.Time
	id         string
	inUse      []string
	kind       string
	migrating  bool
	records    map[string]certificateRecord
	sans       []string
	status     string
	tags       map[string]string
	used       bool
}

func (c *acmCertificate) lookup() bool {
	return c.tags["Purpose"] == lookupPurpose
}

func (c *acmCertificate) site() bool {
	return strings.HasSuffix(dnsName(c.domain), ".convox.site")
}

// keys are the certificate's names without a leading *., which is how ACM shares one record between
// *.example.com and example.com.
func (c *acmCertificate) keys() []string {
	seen := map[string]bool{}
	keys := []string{}

	for _, d := range append([]string{c.domain}, c.sans...) {
		k := certificateKey(d)
		if k != "" && !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}

	return keys
}

func (c *acmCertificate) statusValue() string {
	switch {
	case c.kind != acm.CertificateTypeAmazonIssued:
		return statusImported
	case c.status == acm.CertificateStatusPendingValidation:
		return statusPending
	case c.status == acm.CertificateStatusFailed || c.status == acm.CertificateStatusValidationTimedOut:
		return statusFailed
	case c.migrating:
		return statusMigrating
	case c.email:
		return statusEmail
	default:
		return statusDNS
	}
}

func certificateKey(domain string) string {
	return strings.TrimPrefix(dnsName(domain), "*.")
}

func recordDomain(name string) string {
	n := dnsName(name)

	if i := strings.Index(n, "."); i >= 0 {
		return n[i+1:]
	}

	return n
}

func acmCertificateId(arn string) string {
	parts := strings.Split(arn, "-")
	return fmt.Sprintf("acm-%s", parts[len(parts)-1])
}

// forEach runs fn for every index, at most n at a time. A panic in fn becomes that index's error,
// because a recover only works in the goroutine that panicked.
func forEach(n, count int, fn func(i int) error) []error {
	errs := make([]error, count)
	sem := make(chan struct{}, n)

	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			defer recoverWith(func(err error) { errs[i] = fmt.Errorf("panic: %s", err) })

			errs[i] = fn(i)
		}(i)
	}

	wg.Wait()

	return errs
}

type acmReadOptions struct {
	amazonOnly bool
	first      map[string]bool
	ids        map[string]bool
	skipDNS    bool
	strict     bool
}

// acmCertificates reads every ACM certificate whose state can matter. Failed and timed out
// certificates are read only for their tags, so lookups among them can be swept and hidden.
func (p *Provider) acmCertificates(ctx context.Context, opts acmReadOptions) ([]*acmCertificate, error) {
	summaries, err := p.certificateListACM(acm.KeyAlgorithm_Values())
	if err != nil {
		return nil, err
	}

	certs := []*acmCertificate{}

	for _, s := range summaries {
		arn := aws.StringValue(s.CertificateArn)

		c := &acmCertificate{
			arn:     arn,
			created: aws.TimeValue(s.CreatedAt),
			domain:  aws.StringValue(s.DomainName),
			id:      acmCertificateId(arn),
			kind:    aws.StringValue(s.Type),
			status:  aws.StringValue(s.Status),
			tags:    map[string]string{},
			used:    aws.BoolValue(s.InUse),
		}

		if opts.ids != nil && !opts.ids[c.id] {
			continue
		}

		if opts.amazonOnly && c.kind != acm.CertificateTypeAmazonIssued {
			continue
		}

		switch c.status {
		case acm.CertificateStatusIssued, acm.CertificateStatusPendingValidation, acm.CertificateStatusFailed, acm.CertificateStatusValidationTimedOut:
		default:
			continue
		}

		if opts.skipDNS && certificateSeenDNS(arn) {
			continue
		}

		certs = append(certs, c)
	}

	// A large account cannot be read inside an API deadline, so certificates this rack likely uses
	// are read first.
	sort.SliceStable(certs, func(i, j int) bool {
		return certificateReadRank(certs[i], opts.first) < certificateReadRank(certs[j], opts.first)
	})

	errs := forEach(4, len(certs), func(i int) error {
		return p.acmCertificateRead(ctx, certs[i], opts.strict)
	})

	read := []*acmCertificate{}

	for i, c := range certs {
		if errs[i] == errCertificateGone {
			continue
		}

		c.err = errs[i]
		read = append(read, c)
	}

	return read, nil
}

var errCertificateGone = fmt.Errorf("certificate gone")

func certificateReadRank(c *acmCertificate, first map[string]bool) int {
	switch {
	case first[c.arn]:
		return 0
	case c.used:
		return 1
	case c.status == acm.CertificateStatusPendingValidation:
		return 2
	default:
		return 3
	}
}

var (
	acmPace         = map[string]time.Time{}
	acmPaceInterval = time.Second / 8
	acmPaceLock     sync.Mutex
)

// acmWait spaces calls to one ACM operation across the process to stay under the account's rate
// limit for it, because the SDK backs off at least 10 seconds after a throttled call.
func acmWait(ctx context.Context, op string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	acmPaceLock.Lock()
	at := acmPace[op]
	if now := time.Now(); at.Before(now) {
		at = now
	}
	acmPace[op] = at.Add(acmPaceInterval)
	acmPaceLock.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Until(at)):
		return nil
	}
}

func (p *Provider) acmCertificateRead(ctx context.Context, c *acmCertificate, strict bool) error {
	if err := acmWait(ctx, "ListTagsForCertificate"); err != nil {
		return err
	}

	tres, err := p.acm().ListTagsForCertificateWithContext(ctx, &acm.ListTagsForCertificateInput{CertificateArn: aws.String(c.arn)})
	if awsError(err) == acm.ErrCodeResourceNotFoundException {
		return errCertificateGone
	}
	if err != nil {
		return err
	}

	for _, t := range tres.Tags {
		c.tags[aws.StringValue(t.Key)] = aws.StringValue(t.Value)
	}

	if c.lookup() || c.status == acm.CertificateStatusFailed || c.status == acm.CertificateStatusValidationTimedOut {
		return nil
	}

	if err := acmWait(ctx, "DescribeCertificate"); err != nil {
		return err
	}

	res, err := p.acm().DescribeCertificateWithContext(ctx, &acm.DescribeCertificateInput{CertificateArn: aws.String(c.arn)})
	if awsError(err) == acm.ErrCodeResourceNotFoundException {
		return errCertificateGone
	}
	if err != nil {
		return err
	}

	d := res.Certificate
	if d == nil {
		return errCertificateGone
	}

	c.domain = aws.StringValue(d.DomainName)
	c.expiration = aws.TimeValue(d.NotAfter)
	c.inUse = aws.StringValueSlice(d.InUseBy)
	c.kind = aws.StringValue(d.Type)
	c.used = len(c.inUse) > 0
	c.records = map[string]certificateRecord{}
	c.sans = aws.StringValueSlice(d.SubjectAlternativeNames)
	c.status = aws.StringValue(d.Status)

	for _, o := range d.DomainValidationOptions {
		if o == nil {
			continue
		}

		if aws.StringValue(o.ValidationMethod) == acm.ValidationMethodEmail {
			c.email = true
		}

		if rr := o.ResourceRecord; rr != nil && aws.StringValue(rr.Name) != "" && aws.StringValue(rr.Value) != "" {
			c.records[certificateKey(aws.StringValue(o.DomainName))] = certificateRecord{Name: aws.StringValue(rr.Name), Value: aws.StringValue(rr.Value)}
		}
	}

	if c.kind != acm.CertificateTypeAmazonIssued || c.status != acm.CertificateStatusIssued || !c.email {
		return nil
	}

	if err := acmWait(ctx, "ListCertificateDomainValidations"); err != nil {
		return err
	}

	vs, err := p.acmDomainValidations(ctx, c.arn)
	if awsError(err) == acm.ErrCodeResourceNotFoundException {
		return errCertificateGone
	}
	if err != nil {
		if strict {
			return err
		}
		fmt.Printf("ns=certificates at=validations id=%s error=%q\n", c.id, err)
		return nil
	}

	active := 0

	for _, v := range vs {
		if v == nil {
			continue
		}

		if v.ActiveValidationConfiguration != nil && aws.StringValue(v.ActiveValidationConfiguration.ValidationMethod) == acm.ValidationMethodDns {
			active++
		}

		rv := v.RequestedValidationConfiguration
		if rv == nil || aws.StringValue(rv.ValidationStatus) != acm.DomainStatusPendingValidation {
			continue
		}

		c.migrating = true

		if name, value, ok := rv.record(); ok {
			c.records[certificateKey(aws.StringValue(v.DomainName))] = certificateRecord{Name: name, Value: value}
		}
	}

	if len(vs) > 0 && active == len(vs) {
		c.email = false
		c.migrating = false
	}

	return nil
}

func certificateSeenDNS(arn string) bool {
	certificateState.Lock()
	defer certificateState.Unlock()

	return certificateState.dns[arn]
}

type certificateOwnership struct {
	certs map[string]bool
	gen1  map[string]bool
	lbs   map[string]bool
}

func (p *Provider) certificateOwnership() (*certificateOwnership, error) {
	o := &certificateOwnership{certs: map[string]bool{}, gen1: map[string]bool{}, lbs: map[string]bool{}}

	srs, err := p.listStackResources(p.Rack)
	if err != nil {
		return nil, err
	}

	for _, sr := range srs {
		switch aws.StringValue(sr.ResourceType) {
		case "AWS::CertificateManager::Certificate":
			o.certs[aws.StringValue(sr.PhysicalResourceId)] = true
		case "AWS::ElasticLoadBalancingV2::LoadBalancer":
			o.lbs[aws.StringValue(sr.PhysicalResourceId)] = true
		}
	}

	stacks, err := p.describeStacks(&cloudformation.DescribeStacksInput{})
	if err != nil {
		return nil, err
	}

	for _, stack := range stacks {
		tags := stackTags(stack)

		if tags["System"] != "convox" || tags["Type"] != "app" || tags["Rack"] != p.Rack || coalesces(tags["Generation"], "1") != "1" {
			continue
		}

		for _, v := range stackParameters(stack) {
			for _, part := range strings.Split(v, ",") {
				if strings.HasPrefix(part, "arn:") && strings.Contains(part, ":acm:") {
					o.gen1[part] = true
				}
			}
		}
	}

	return o, nil
}

func (o *certificateOwnership) likely() map[string]bool {
	m := map[string]bool{}

	for arn := range o.certs {
		m[arn] = true
	}

	for arn := range o.gen1 {
		m[arn] = true
	}

	return m
}

func (o *certificateOwnership) owns(c *acmCertificate, rack string) bool {
	if c.lookup() || c.kind != acm.CertificateTypeAmazonIssued {
		return false
	}

	if c.site() {
		return o.certs[c.arn] || c.tags["Rack"] == rack
	}

	if c.tags["Rack"] == rack || o.gen1[c.arn] {
		return true
	}

	for _, u := range c.inUse {
		if o.lbs[u] {
			return true
		}
	}

	return false
}
