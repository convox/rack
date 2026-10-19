package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/convox/rack/pkg/structs"
)

const (
	resultDNS        = "dns"
	resultError      = "error"
	resultFailed     = "failed"
	resultImported   = "imported"
	resultPending    = "pending"
	resultPreparing  = "preparing"
	resultPublishing = "publishing"
	resultSwitching  = "switching"
	resultWaiting    = "waiting"

	statusDNS       = "dns"
	statusEmail     = "email"
	statusFailed    = "failed"
	statusImported  = "imported"
	statusMigrating = "migrating"
	statusPending   = "pending"

	certificateCacheKey = "certificates/validation.json"
	lookupPurpose       = "validation-lookup"
)

var (
	certificateAPIDeadline   = 40 * time.Second
	certificateAuthCutoff    = 25 * time.Second
	certificateDeleteWait    = 10 * time.Second
	certificateLogInterval   = time.Hour
	certificateLookupBackoff = 6 * time.Hour
	certificateLookupCutoff  = 15 * time.Second
	certificateRecordTick    = 2 * time.Second
	certificateRecordWait    = 30 * time.Second
	certificateSweepAge      = 10 * time.Minute
	certificateWorkerTick    = 5 * time.Minute
)

type certificateRecord struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type certificateCacheError struct {
	Error string    `json:"error"`
	Time  time.Time `json:"time"`
}

type certificateCache struct {
	Errors  map[string]certificateCacheError `json:"errors"`
	Records map[string]certificateRecord     `json:"records"`
}

type registrarHold struct {
	err   error
	until time.Time
}

// certificateState is shared by every Provider copy in the process, so it outlives a single pass.
var certificateState = struct {
	sync.Mutex
	dns     map[string]bool
	holds   map[string]registrarHold
	logged  map[string]time.Time
	records map[string]certificateRecord
}{
	dns:     map[string]bool{},
	holds:   map[string]registrarHold{},
	logged:  map[string]time.Time{},
	records: map[string]certificateRecord{},
}

var certificateCacheLock sync.Mutex

var certificateLogf = func(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

type certificateRun struct {
	act      bool
	cache    *certificateCache
	ctx      context.Context
	deadline time.Time
	dns      *dnsChecker
	own      *certificateOwnership
	p        *Provider
	worker   bool

	mu     sync.Mutex
	hosted map[string]string
}

func (p *Provider) certificateRun(ctx context.Context, deadline time.Time, act, worker bool) (*certificateRun, error) {
	own, err := p.certificateOwnership()
	if err != nil {
		return nil, err
	}

	cache, err := p.certificateCacheLoad()
	if err != nil {
		return nil, err
	}

	authUntil := time.Time{}
	if !act {
		authUntil = deadline.Add(certificateAuthCutoff - certificateAPIDeadline)
	}

	return &certificateRun{
		act:      act,
		cache:    cache,
		ctx:      ctx,
		deadline: deadline,
		dns:      newDNSChecker(authUntil),
		hosted:   map[string]string{},
		own:      own,
		p:        p,
		worker:   worker,
	}, nil
}

func (p *Provider) certificateCacheLoad() (*certificateCache, error) {
	c := &certificateCache{Errors: map[string]certificateCacheError{}, Records: map[string]certificateRecord{}}

	exists, err := p.s3Exists(p.SettingsBucket, certificateCacheKey)
	if err != nil {
		return nil, err
	}

	if exists {
		data, err := p.s3Get(p.SettingsBucket, certificateCacheKey)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(data, c); err != nil {
			return nil, err
		}

		if c.Errors == nil {
			c.Errors = map[string]certificateCacheError{}
		}

		if c.Records == nil {
			c.Records = map[string]certificateRecord{}
		}
	}

	certificateState.Lock()
	defer certificateState.Unlock()

	for k, r := range certificateState.records {
		c.Records[k] = r
	}

	return c, nil
}

// certificateCacheSave re-reads the object before writing, so two processes lose at most each
// other's latest change.
func (p *Provider) certificateCacheSave(update func(c *certificateCache)) error {
	certificateCacheLock.Lock()
	defer certificateCacheLock.Unlock()

	c, err := p.certificateCacheLoad()
	if err != nil {
		return err
	}

	update(c)

	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return p.s3Put(p.SettingsBucket, certificateCacheKey, data, false)
}

func (r *certificateRun) record(key string) (certificateRecord, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	rec, ok := r.cache.Records[key]

	return rec, ok
}

// learn keeps records in memory before writing them, so a failed write never repeats a lookup.
func (r *certificateRun) learn(records map[string]certificateRecord) {
	changed := map[string]certificateRecord{}

	r.mu.Lock()
	for k, rec := range records {
		if r.cache.Records[k] != rec {
			r.cache.Records[k] = rec
			changed[k] = rec
		}
	}
	r.mu.Unlock()

	if len(changed) == 0 {
		return
	}

	certificateState.Lock()
	for k, rec := range changed {
		certificateState.records[k] = rec
	}
	certificateState.Unlock()

	if !r.act {
		return
	}

	if err := r.p.certificateCacheSave(func(c *certificateCache) {
		for k, rec := range changed {
			c.Records[k] = rec
		}
	}); err != nil {
		fmt.Printf("ns=certificates at=cache error=%q\n", err)
	}
}

func (r *certificateRun) lookupError(c *acmCertificate, lerr error) {
	r.mu.Lock()
	if lerr == nil {
		if _, ok := r.cache.Errors[c.id]; !ok {
			r.mu.Unlock()
			return
		}
		delete(r.cache.Errors, c.id)
	} else {
		r.cache.Errors[c.id] = certificateCacheError{Error: lerr.Error(), Time: time.Now().UTC()}
	}
	r.mu.Unlock()

	if err := r.p.certificateCacheSave(func(cc *certificateCache) {
		if lerr == nil {
			delete(cc.Errors, c.id)
		} else {
			cc.Errors[c.id] = certificateCacheError{Error: lerr.Error(), Time: time.Now().UTC()}
		}
	}); err != nil {
		fmt.Printf("ns=certificates at=cache error=%q\n", err)
	}
}

func (r *certificateRun) records(c *acmCertificate) ([]certificateRecord, []string) {
	recs := []certificateRecord{}
	missing := []string{}

	for _, k := range c.keys() {
		if rec, ok := c.records[k]; ok {
			recs = append(recs, rec)
			continue
		}

		if rec, ok := r.record(k); ok && c.email && !c.migrating {
			recs = append(recs, rec)
			continue
		}

		missing = append(missing, k)
	}

	return recs, missing
}

func (r *certificateRun) seed(certs []*acmCertificate) {
	learned := map[string]certificateRecord{}

	for _, c := range certs {
		if c.lookup() || c.kind != acm.CertificateTypeAmazonIssued || (c.email && !c.migrating) {
			continue
		}

		for k, rec := range c.records {
			learned[k] = rec
		}
	}

	r.learn(learned)
}

func (r *certificateRun) result(c *acmCertificate) structs.CertificateMigration {
	m := structs.CertificateMigration{Id: c.id, Domain: c.domain}

	switch c.statusValue() {
	case statusImported:
		m.Result = resultImported
		return m
	case statusFailed:
		m.Result = resultFailed
		return m
	case statusDNS:
		m.Result = resultDNS
		return m
	case statusPending:
		if c.email {
			m.Result = resultPending
			return m
		}
	}

	recs, missing := r.records(c)
	if len(missing) > 0 {
		m.Result = resultPreparing

		r.mu.Lock()
		e, failed := r.cache.Errors[c.id]
		r.mu.Unlock()

		if failed && !r.act {
			m.Result, m.Detail = resultError, e.Error
		}

		return m
	}

	m.Result, m.Detail = r.ensure(c, recs)

	if m.Result != resultSwitching {
		return m
	}

	switch c.statusValue() {
	case statusPending:
		m.Result = resultPending
	case statusEmail:
		if r.act {
			if err := r.switchToDNS(c); err != nil {
				m.Result, m.Detail = resultError, err.Error()
			}
		}
	}

	return m
}

func (r *certificateRun) ensure(c *acmCertificate, recs []certificateRecord) (string, string) {
	results := make([]string, len(recs))
	details := make([]string, len(recs))

	errs := forEach(8, len(recs), func(i int) error {
		results[i], details[i] = r.ensureRecord(c, recs[i])
		return nil
	})

	result, detail := resultSwitching, ""

	rank := map[string]int{resultSwitching: 0, resultPublishing: 1, resultWaiting: 2, resultError: 3}

	for i := range recs {
		if errs[i] != nil {
			results[i], details[i] = resultError, errs[i].Error()
		}

		if rank[results[i]] > rank[result] {
			result, detail = results[i], details[i]
		}
	}

	return result, detail
}

func (r *certificateRun) ensureRecord(c *acmCertificate, rec certificateRecord) (string, string) {
	z := r.dns.zone(r.ctx, recordDomain(rec.Name))

	check, answered := r.dns.authoritative(r.ctx, z, rec.Name, rec.Value)

	var perr error
	published := false

	// The VPC resolver would cache the load balancer wildcard's answer for an unpublished name for a day.
	if !answered && c.site() && r.act {
		perr, published = r.publish(c, rec), true
	}

	if !answered && c.site() && !r.act {
		return resultPublishing, ""
	}

	if !answered {
		check = r.dns.resolver(r.ctx, rec.Name, rec.Value)
	}

	if check == checkOK {
		return resultSwitching, ""
	}

	if c.site() {
		if r.act && !published {
			perr = r.publish(c, rec)
		}

		if perr != nil {
			return resultError, perr.Error()
		}

		return resultPublishing, ""
	}

	if !r.p.Route53CertificateValidation {
		return resultWaiting, ""
	}

	id := r.hostedZone(z)
	if id == "" {
		return resultWaiting, ""
	}

	if !r.act {
		return resultPublishing, ""
	}

	if !r.route53Write(c, id, rec) {
		return resultWaiting, ""
	}

	return resultPublishing, ""
}

func (r *certificateRun) publish(c *acmCertificate, rec certificateRecord) error {
	domain := c.domain

	for _, d := range append([]string{c.domain}, c.sans...) {
		if certificateKey(d) == recordDomain(rec.Name) {
			domain = d
			break
		}
	}

	if r.worker {
		certificateState.Lock()
		h, held := certificateState.holds[rec.Name]
		certificateState.Unlock()

		if held && time.Now().Before(h.until) {
			return h.err
		}
	}

	err := r.p.registrarPublish(r.ctx, domain, rec.Name, rec.Value)
	if err == nil {
		return nil
	}

	re, ok := err.(registrarError)

	if ok && re.support() {
		err = fmt.Errorf("%s, contact Convox support", re.Error())
	}

	if r.worker && ok && !re.retry() {
		certificateState.Lock()
		certificateState.holds[rec.Name] = registrarHold{err: err, until: time.Now().Add(certificateLogInterval)}
		certificateState.Unlock()
	}

	r.logRack("registrar:"+rec.Name, fmt.Sprintf("certificate %s (%s) record %s CNAME %s: %s", c.id, c.domain, rec.Name, rec.Value, err))

	return err
}

func (r *certificateRun) switchToDNS(c *acmCertificate) error {
	err := r.p.acmUpdateValidationDNS(r.ctx, c.arn)
	if awsError(err) == acm.ErrCodeConflictException {
		return nil
	}
	if err != nil {
		return err
	}

	if r.worker {
		line := fmt.Sprintf("certificate %s (%s) switched to DNS validation", c.id, c.domain)
		certificateLogf("%s", line)
		r.logApp(c, "", line)
	}

	vs, err := r.p.acmDomainValidations(r.ctx, c.arn)
	if err != nil {
		return nil
	}

	changed := map[string]certificateRecord{}

	for _, v := range vs {
		if v == nil {
			continue
		}

		name, value, ok := v.RequestedValidationConfiguration.record()
		if !ok {
			continue
		}

		k := certificateKey(aws.StringValue(v.DomainName))
		rec := certificateRecord{Name: name, Value: value}

		if cached, ok := r.record(k); !ok || cached != rec {
			changed[k] = rec
		}
	}

	r.learn(changed)

	if c.site() {
		for _, rec := range changed {
			if err := r.publish(c, rec); err != nil {
				return err
			}
		}
	}

	return nil
}

// lookup learns the records of an email certificate from a throwaway DNS request for the same
// names, which ACM answers with the records for this account.
func (r *certificateRun) lookup(c *acmCertificate) error {
	if !r.worker && time.Until(r.deadline) < certificateLookupCutoff {
		return errLookupDeferred
	}

	sans := []string{}
	seen := map[string]bool{certificateKey(c.domain): true}

	_, missing := r.records(c)
	need := map[string]bool{}

	for _, k := range missing {
		need[k] = true
	}

	for _, s := range c.sans {
		k := certificateKey(s)
		if seen[k] || !need[k] {
			continue
		}

		seen[k] = true
		sans = append(sans, s)
	}

	req := &acm.RequestCertificateInput{
		DomainName:       aws.String(c.domain),
		IdempotencyToken: aws.String(randomString(32)),
		Tags: []*acm.Tag{
			{Key: aws.String("System"), Value: aws.String("convox")},
			{Key: aws.String("Rack"), Value: aws.String(r.p.Rack)},
			{Key: aws.String("Purpose"), Value: aws.String(lookupPurpose)},
		},
		ValidationMethod: aws.String(acm.ValidationMethodDns),
	}

	if len(sans) > 0 {
		req.SubjectAlternativeNames = aws.StringSlice(sans)
	}

	res, err := r.p.acm().RequestCertificateWithContext(r.ctx, req)
	if err != nil {
		return err
	}

	arn := aws.StringValue(res.CertificateArn)

	defer r.p.lookupDelete(arn)

	learned, err := r.p.certificateRecordsWait(r.ctx, arn, certificateRecordWait)
	if err != nil {
		return err
	}

	r.learn(learned)

	return nil
}

var errLookupDeferred = fmt.Errorf("lookup deferred")

func (p *Provider) certificateRecordsWait(ctx context.Context, arn string, wait time.Duration) (map[string]certificateRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()

	for {
		res, err := p.acm().DescribeCertificateWithContext(ctx, &acm.DescribeCertificateInput{CertificateArn: aws.String(arn)})
		if err != nil {
			return nil, err
		}

		if res.Certificate != nil && aws.StringValue(res.Certificate.Status) == acm.CertificateStatusFailed {
			return nil, fmt.Errorf("validation failed: %s", aws.StringValue(res.Certificate.FailureReason))
		}

		records := map[string]certificateRecord{}
		complete := res.Certificate != nil && len(res.Certificate.DomainValidationOptions) > 0

		if res.Certificate != nil {
			for _, o := range res.Certificate.DomainValidationOptions {
				rr := o.ResourceRecord
				if rr == nil || aws.StringValue(rr.Name) == "" || aws.StringValue(rr.Value) == "" {
					complete = false
					continue
				}

				records[certificateKey(aws.StringValue(o.DomainName))] = certificateRecord{Name: aws.StringValue(rr.Name), Value: aws.StringValue(rr.Value)}
			}
		}

		if complete {
			return records, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("records not ready")
		case <-time.After(certificateRecordTick):
		}
	}
}

// lookupDelete has its own deadline, so a lookup is removed even after the pass ran out of time.
func (p *Provider) lookupDelete(arn string) {
	ctx, cancel := context.WithTimeout(context.Background(), certificateDeleteWait)
	defer cancel()

	tres, err := p.acm().ListTagsForCertificateWithContext(ctx, &acm.ListTagsForCertificateInput{CertificateArn: aws.String(arn)})
	if err != nil {
		fmt.Printf("ns=certificates at=lookup.delete arn=%s error=%q\n", arn, err)
		return
	}

	purpose := ""

	for _, t := range tres.Tags {
		if aws.StringValue(t.Key) == "Purpose" {
			purpose = aws.StringValue(t.Value)
		}
	}

	if purpose != lookupPurpose {
		return
	}

	res, err := p.acm().DescribeCertificateWithContext(ctx, &acm.DescribeCertificateInput{CertificateArn: aws.String(arn)})
	if err != nil {
		fmt.Printf("ns=certificates at=lookup.delete arn=%s error=%q\n", arn, err)
		return
	}

	if res.Certificate == nil || len(res.Certificate.InUseBy) > 0 {
		return
	}

	if _, err := p.acm().DeleteCertificateWithContext(ctx, &acm.DeleteCertificateInput{CertificateArn: aws.String(arn)}); err != nil && awsError(err) != acm.ErrCodeResourceNotFoundException {
		fmt.Printf("ns=certificates at=lookup.delete arn=%s error=%q\n", arn, err)
	}
}

func (r *certificateRun) sweep(certs []*acmCertificate) {
	for _, c := range certs {
		if !c.lookup() || c.tags["Rack"] != r.p.Rack || c.used || time.Since(c.created) < certificateSweepAge {
			continue
		}

		r.p.lookupDelete(c.arn)
	}
}

func certificateLogDue(key string) bool {
	certificateState.Lock()
	defer certificateState.Unlock()

	if time.Since(certificateState.logged[key]) < certificateLogInterval {
		return false
	}

	certificateState.logged[key] = time.Now()

	return true
}

func (r *certificateRun) logRack(key, line string) {
	if r.worker && certificateLogDue(key) {
		certificateLogf("%s", line)
	}
}

func (r *certificateRun) logApp(c *acmCertificate, key, line string) {
	app := c.tags["Name"]

	if !r.worker || app == "" || c.tags["Rack"] != r.p.Rack || (key != "" && !certificateLogDue(key)) {
		return
	}

	group, err := r.p.getStackLogGroup(r.p.rackStack(app))
	if err != nil {
		return
	}

	req := &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(group),
		LogStreamName: aws.String("system/certificates"),
		LogEvents: []*cloudwatchlogs.InputLogEvent{{
			Message:   aws.String(line),
			Timestamp: aws.Int64(time.Now().UnixNano() / int64(time.Millisecond)),
		}},
	}

	if _, err := r.p.putLogEvents(req); err != nil {
		fmt.Printf("ns=certificates at=log app=%s error=%q\n", app, err)
	}
}

func (p *Provider) certificatePass(ids []string, deadline time.Time, worker bool) (structs.CertificateMigrations, error) {
	ctx, cancel := context.WithDeadline(p.Context(), deadline)
	defer cancel()

	return p.certificateMigrations(ctx, ids, deadline, true, worker)
}

func (p *Provider) certificateMigrations(ctx context.Context, ids []string, deadline time.Time, act, worker bool) (structs.CertificateMigrations, error) {
	r, err := p.certificateRun(ctx, deadline, act, worker)
	if err != nil {
		return nil, err
	}

	opts := acmReadOptions{amazonOnly: ids == nil, first: r.own.likely(), skipDNS: worker, strict: true}

	named := map[string]bool{}

	if ids != nil {
		opts.ids = map[string]bool{}

		for _, id := range ids {
			opts.ids[id] = true
			named[id] = true
		}
	}

	certs, err := p.acmCertificates(ctx, opts)
	if err != nil {
		return nil, err
	}

	r.seed(certs)

	if act {
		r.sweep(certs)
	}

	ms := structs.CertificateMigrations{}
	owned := []*acmCertificate{}

	for _, c := range certs {
		if c.lookup() {
			continue
		}

		delete(named, c.id)

		if c.err != nil {
			if ids != nil || r.own.owns(c, p.Rack) {
				ms = append(ms, r.cut(certificateError(c, c.err)))
			}
			continue
		}

		if worker && c.statusValue() == statusDNS {
			certificateState.Lock()
			certificateState.dns[c.arn] = true
			certificateState.Unlock()
		}

		switch {
		case c.kind != acm.CertificateTypeAmazonIssued:
			ms = append(ms, structs.CertificateMigration{Id: c.id, Domain: c.domain, Result: resultImported})
		case !r.own.owns(c, p.Rack):
			if ids != nil {
				ms = append(ms, structs.CertificateMigration{Id: c.id, Domain: c.domain, Result: resultError, Detail: "not used by this rack"})
			}
		default:
			owned = append(owned, c)
		}
	}

	for id := range named {
		ms = append(ms, structs.CertificateMigration{Id: id, Result: resultError, Detail: "certificate not found"})
	}

	sort.SliceStable(owned, func(i, j int) bool { return certificateOrder(owned[i]) < certificateOrder(owned[j]) })

	results := r.pass(owned)

	if worker {
		r.logWaiting(owned, results)
	}

	for _, m := range results {
		ms = append(ms, m)
	}

	sort.Slice(ms, func(i, j int) bool { return ms[i].Id < ms[j].Id })

	return ms, nil
}

func certificateError(c *acmCertificate, err error) structs.CertificateMigration {
	return structs.CertificateMigration{Id: c.id, Domain: c.domain, Result: resultError, Detail: err.Error()}
}

// pass gets every result, then runs lookups, then redoes the results that waited on one. Each step
// goes group by group, so a load balancer's name reaches the registrar before its services' names.
func (r *certificateRun) pass(owned []*acmCertificate) map[string]structs.CertificateMigration {
	out := map[string]structs.CertificateMigration{}
	retry := []*acmCertificate{}

	for _, g := range certificateGroups(owned) {
		for i, m := range r.results(g) {
			out[g[i].id] = m

			if r.act && m.Result == resultPreparing && g[i].statusValue() == statusEmail {
				retry = append(retry, g[i])
			}
		}
	}

	failed := r.lookups(retry)

	for _, g := range certificateGroups(retry) {
		for i, m := range r.results(g) {
			if err, ok := failed[g[i].id]; ok {
				m = certificateError(g[i], err)
			}

			out[g[i].id] = m
		}
	}

	return out
}

func (r *certificateRun) results(certs []*acmCertificate) []structs.CertificateMigration {
	rs := make([]structs.CertificateMigration, len(certs))

	errs := forEach(4, len(certs), func(i int) error {
		rs[i] = r.cut(r.result(certs[i]))
		return nil
	})

	for i, c := range certs {
		if errs[i] != nil {
			rs[i] = r.cut(certificateError(c, errs[i]))
		}
	}

	return rs
}

// cut reports work the deadline stopped as preparing, which the next worker pass finishes.
func (r *certificateRun) cut(m structs.CertificateMigration) structs.CertificateMigration {
	if r.ctx.Err() != nil && (m.Result == resultError || m.Result == resultWaiting) {
		m.Result, m.Detail = resultPreparing, ""
	}

	return m
}

// lookups runs one lookup for each set of names no other lookup covers. The worker skips a
// certificate whose lookup failed recently, so a lookup that keeps failing cannot use up the
// account's yearly ACM request quota.
func (r *certificateRun) lookups(certs []*acmCertificate) map[string]error {
	failed := map[string]error{}
	planned := map[string]bool{}
	todo := []*acmCertificate{}

	for _, c := range certs {
		r.mu.Lock()
		e, ok := r.cache.Errors[c.id]
		r.mu.Unlock()

		if ok && r.worker && time.Since(e.Time) < certificateLookupBackoff {
			failed[c.id] = fmt.Errorf("%s", e.Error)
			continue
		}

		_, missing := r.records(c)

		covered := true

		for _, k := range missing {
			if !planned[k] {
				covered = false
			}
		}

		if covered {
			continue
		}

		for _, k := range missing {
			planned[k] = true
		}

		todo = append(todo, c)
	}

	errs := forEach(4, len(todo), func(i int) error {
		c := todo[i]

		err := r.lookup(c)
		if err == errLookupDeferred || (err != nil && r.ctx.Err() != nil) {
			return nil
		}

		r.lookupError(c, err)

		if err != nil {
			r.logRack("lookup:"+c.id, fmt.Sprintf("certificate %s (%s) lookup failed: %s", c.id, c.domain, err))
		}

		return err
	})

	for i, c := range todo {
		if errs[i] != nil {
			failed[c.id] = errs[i]
		}
	}

	return failed
}

func (r *certificateRun) logWaiting(owned []*acmCertificate, results map[string]structs.CertificateMigration) {
	waiting := 0

	for _, c := range owned {
		if c.site() || results[c.id].Result != resultWaiting {
			continue
		}

		waiting++

		if c.statusValue() == statusPending {
			r.logApp(c, "app:"+c.id, fmt.Sprintf("certificate %s (%s) is waiting for its DNS record: run convox certs dns", c.id, c.domain))
		} else {
			r.logApp(c, "app:"+c.id, fmt.Sprintf("certificate %s (%s) needs a DNS record before 2027-09-30: run convox certs dns", c.id, c.domain))
		}
	}

	if waiting == 0 {
		return
	}

	noun := "certificates need DNS records"
	if waiting == 1 {
		noun = "certificate needs DNS records"
	}

	r.logRack("rack:waiting", fmt.Sprintf("%d %s: run convox certs dns", waiting, noun))
}

// certificateOrder puts certificates with only load balancer convox.site names before ones with
// service names, because the registrar refuses a service name under a load balancer it has not
// registered yet.
func certificateOrder(c *acmCertificate) int {
	if !c.site() {
		return 2
	}

	for _, k := range c.keys() {
		if strings.Count(k, ".") > 3 {
			return 1
		}
	}

	return 0
}

func certificateGroups(certs []*acmCertificate) [][]*acmCertificate {
	groups := [][]*acmCertificate{}

	for i, c := range certs {
		if i == 0 || certificateOrder(c) != certificateOrder(certs[i-1]) {
			groups = append(groups, []*acmCertificate{})
		}

		groups[len(groups)-1] = append(groups[len(groups)-1], c)
	}

	return groups
}

func (p *Provider) CertificateDns() (*structs.CertificateDns, error) {
	deadline := time.Now().Add(certificateAPIDeadline)

	ctx, cancel := context.WithDeadline(p.Context(), deadline)
	defer cancel()

	r, err := p.certificateRun(ctx, deadline, false, false)
	if err != nil {
		return nil, err
	}

	certs, err := p.acmCertificates(ctx, acmReadOptions{amazonOnly: true, first: r.own.likely()})
	if err != nil {
		return nil, err
	}

	r.seed(certs)

	out := &structs.CertificateDns{Errors: []structs.CertificateDnsError{}, Preparing: []string{}, Records: []structs.CertificateDnsRecord{}}
	rows := map[string]*structs.CertificateDnsRecord{}
	order := []string{}

	for _, c := range certs {
		if c.err != nil {
			switch {
			case c.site() || c.lookup():
			case ctx.Err() != nil:
				out.Preparing = append(out.Preparing, c.id)
			case r.own.owns(c, p.Rack):
				out.Errors = append(out.Errors, structs.CertificateDnsError{Id: c.id, Error: c.err.Error()})
			default:
				fmt.Printf("ns=certificates at=dns id=%s error=%q\n", c.id, c.err)
			}
			continue
		}

		if c.site() || !r.own.owns(c, p.Rack) {
			continue
		}

		switch c.statusValue() {
		case statusFailed, statusImported:
			continue
		case statusPending:
			if c.email {
				continue
			}
		}

		recs, missing := r.records(c)

		if len(missing) > 0 {
			if e, ok := r.cache.Errors[c.id]; ok {
				out.Errors = append(out.Errors, structs.CertificateDnsError{Id: c.id, Error: e.Error})
			} else {
				out.Preparing = append(out.Preparing, c.id)
			}
			continue
		}

		for _, rec := range recs {
			name := dnsName(rec.Name)

			row, ok := rows[name]
			if !ok {
				row = &structs.CertificateDnsRecord{Name: rec.Name, Type: route53.RRTypeCname, Value: rec.Value, Certificates: []string{}}
				rows[name] = row
				order = append(order, name)
			}

			row.Certificates = append(row.Certificates, c.id)
		}
	}

	forEach(8, len(order), func(i int) error {
		row := rows[order[i]]

		row.Zone = recordDomain(row.Name)

		if ctx.Err() != nil {
			row.Check = checkUnknown
			return nil
		}

		z := r.dns.zone(ctx, recordDomain(row.Name))

		if z.name != "" {
			row.Zone = z.name
		}

		row.Check = r.dns.check(ctx, z, row.Name, row.Value)
		row.Route53 = !p.Route53CertificateValidation && r.hostedZone(z) != ""

		return nil
	})

	for _, name := range order {
		row := rows[name]

		if row.Check == "" {
			row.Check = checkUnknown
		}

		sort.Strings(row.Certificates)
		out.Records = append(out.Records, *row)
	}

	sort.SliceStable(out.Records, func(i, j int) bool {
		if out.Records[i].Zone != out.Records[j].Zone {
			return out.Records[i].Zone < out.Records[j].Zone
		}
		return out.Records[i].Name < out.Records[j].Name
	})

	sort.Strings(out.Preparing)
	sort.Slice(out.Errors, func(i, j int) bool { return out.Errors[i].Id < out.Errors[j].Id })

	return out, nil
}

func (p *Provider) CertificateMigrate(ids []string) (structs.CertificateMigrations, error) {
	return p.certificatePass(ids, time.Now().Add(certificateAPIDeadline), false)
}

func (p *Provider) CertificateMigrationList(ids []string) (structs.CertificateMigrations, error) {
	deadline := time.Now().Add(certificateAPIDeadline)

	ctx, cancel := context.WithDeadline(p.Context(), deadline)
	defer cancel()

	return p.certificateMigrations(ctx, ids, deadline, false, false)
}
