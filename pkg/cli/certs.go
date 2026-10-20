package cli

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
)

func init() {
	register("certs", "list certificates", Certs, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Validate: stdcli.Args(0),
	})

	register("certs delete", "delete a certificate", CertsDelete, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagRack},
		Usage:    "<cert>",
		Validate: stdcli.Args(1),
	})

	register("certs dns", "list the dns records for certificates", CertsDns, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagRack,
			stdcli.BoolFlag("zone-file", "", "print records as zone file lines"),
		},
		Validate: stdcli.Args(0),
	})

	register("certs generate", "generate a certificate", CertsGenerate, stdcli.CommandOptions{
		Flags:    []stdcli.Flag{flagId, flagRack},
		Usage:    "<domain> [domain...]",
		Validate: stdcli.ArgsMin(1),
	})

	register("certs import", "import a certificate", CertsImport, stdcli.CommandOptions{
		Flags: []stdcli.Flag{
			flagId,
			flagRack,
			stdcli.StringFlag("chain", "", "intermediate certificate chain"),
		},
		Usage:    "<pub> <key>",
		Validate: stdcli.Args(2),
	})

	register("certs migrate", "switch certificates to dns validation", CertsMigrate, stdcli.CommandOptions{
		Flags: []stdcli.Flag{flagRack, flagWait},
		Usage: "[cert...]",
	})
}

func Certs(rack sdk.Interface, c *stdcli.Context) error {
	cs, err := rack.CertificateList()
	if err != nil {
		return err
	}

	t := c.Table("ID", "DOMAIN", "EXPIRES", "STATUS")

	for _, c := range cs {
		t.AddRow(c.Id, c.Domain, helpers.Ago(c.Expiration), c.Status)
	}

	if err := t.Print(); err != nil {
		return err
	}

	if lines := certsEmailLines(cs); len(lines) > 0 {
		return certsWrite(c.Writer().Stdout, append([]string{""}, lines...)...)
	}

	return nil
}

func CertsDelete(rack sdk.Interface, c *stdcli.Context) error {
	cert := c.Arg(0)

	c.Startf("Deleting certificate <id>%s</id>", cert)

	if err := rack.CertificateDelete(cert); err != nil {
		return err
	}

	return c.OK()
}

func CertsDns(rack sdk.Interface, c *stdcli.Context) error {
	d, err := rack.CertificateDns()
	if err != nil {
		return certsRackError("dns", err)
	}

	if d == nil {
		d = &structs.CertificateDns{}
	}

	if c.Bool("zone-file") {
		return certsZoneFile(c, d)
	}

	cs, err := rack.CertificateList()
	if err != nil {
		return err
	}

	lines := append([]string{certsDnsSummary(d.Records)}, certsDnsLines(d, cs)...)

	if len(d.Records) > 0 {
		t := c.Table("ZONE", "NAME", "TYPE", "VALUE", "CHECK", "CERTIFICATES")

		for _, r := range d.Records {
			t.AddRow(r.Zone, r.Name, r.Type, r.Value, r.Check, strings.Join(r.Certificates, ","))
		}

		if err := t.Print(); err != nil {
			return err
		}

		lines = append([]string{""}, lines...)
	}

	return certsWrite(c.Writer().Stdout, lines...)
}

func CertsGenerate(rack sdk.Interface, c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	c.Startf("Generating certificate")

	cr, err := rack.CertificateGenerate(c.Args, structs.CertificateGenerateOptions{Validation: options.String("dns")})
	if err != nil {
		return err
	}

	c.OK(cr.Id)

	if err := certsGenerateRecords(rack, c, cr.Id); err != nil {
		return err
	}

	if c.Bool("id") {
		fmt.Fprint(stdout, cr.Id)
	}

	return nil
}

func CertsImport(rack sdk.Interface, c *stdcli.Context) error {
	var stdout io.Writer

	if c.Bool("id") {
		stdout = c.Writer().Stdout
		c.Writer().Stdout = c.Writer().Stderr
	}

	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	pub, err := os.ReadFile(c.Arg(0))
	if err != nil {
		return err
	}

	key, err := os.ReadFile(c.Arg(1))
	if err != nil {
		return err
	}

	var opts structs.CertificateCreateOptions

	if cf := c.String("chain"); cf != "" {
		chain, err := os.ReadFile(cf)
		if err != nil {
			return err
		}

		opts.Chain = options.String(string(chain))
	}

	c.Startf("Importing certificate")

	var cr *structs.Certificate

	if s.Version <= "20180708231844" {
		cr, err = rack.CertificateCreateClassic(string(pub), string(key), opts)
		if err != nil {
			return err
		}
	} else {
		cr, err = rack.CertificateCreate(string(pub), string(key), opts)
		if err != nil {
			return err
		}
	}

	c.OK(cr.Id)

	if c.Bool("id") {
		fmt.Fprint(stdout, cr.Id)
	}

	return nil
}

func CertsMigrate(rack sdk.Interface, c *stdcli.Context) error {
	var ids []string

	if len(c.Args) > 0 {
		ids = c.Args
	}

	ms, err := rack.CertificateMigrate(ids)
	if err != nil {
		return certsRackError("migrate", err)
	}

	if err := certsMigrationTable(c, ms); err != nil {
		return err
	}

	if !c.Bool("wait") {
		return nil
	}

	return certsMigrateWait(rack, c, ms)
}

func certsEmailLines(cs structs.Certificates) []string {
	custom, site := 0, 0

	for _, cert := range cs {
		if !cert.Owned || cert.Status != "email" {
			continue
		}

		if certsConvoxSite(cert.Domain) {
			site++
		} else {
			custom++
		}
	}

	lines := []string{}

	switch {
	case custom == 1:
		lines = append(lines, "1 certificate uses email validation, which stops renewing on 2027-09-30: run convox certs dns")
	case custom > 1:
		lines = append(lines, fmt.Sprintf("%d certificates use email validation, which stops renewing on 2027-09-30: run convox certs dns", custom))
	}

	switch {
	case site == 1:
		lines = append(lines, "1 convox.site certificate is still on email validation: run convox certs migrate to see why")
	case site > 1:
		lines = append(lines, fmt.Sprintf("%d convox.site certificates are still on email validation: run convox certs migrate to see why", site))
	}

	return lines
}

func certsConvoxSite(domain string) bool {
	return strings.HasSuffix(strings.TrimPrefix(strings.ToLower(domain), "*."), ".convox.site")
}

func certsDnsSummary(rs []structs.CertificateDnsRecord) string {
	summary := fmt.Sprintf("%d records", len(rs))

	if len(rs) == 1 {
		summary = "1 record"
	}

	counts := []string{}

	for _, check := range []string{"ok", "missing", "wrong", "unknown"} {
		n := 0

		for _, r := range rs {
			if r.Check == check {
				n++
			}
		}

		if n > 0 {
			counts = append(counts, fmt.Sprintf("%d %s", n, check))
		}
	}

	if len(counts) == 0 {
		return summary
	}

	return fmt.Sprintf("%s: %s", summary, strings.Join(counts, ", "))
}

func certsDnsLines(d *structs.CertificateDns, cs structs.Certificates) []string {
	lines := []string{}
	zones := map[string]bool{}

	for _, r := range d.Records {
		if r.Route53 && !zones[r.Zone] {
			zones[r.Zone] = true
			lines = append(lines, fmt.Sprintf("%s is a Route 53 zone in this AWS account. To have the rack add its records: convox rack params set Route53CertificateValidation=Yes", r.Zone))
		}
	}

	lines = append(lines, certsDnsNotes(d)...)

	unused := []string{}

	for _, cert := range cs {
		if !cert.Owned && cert.Status == "email" && !certsConvoxSite(cert.Domain) {
			unused = append(unused, cert.Id)
		}
	}

	switch {
	case len(unused) == 1:
		lines = append(lines, fmt.Sprintf("%s uses email validation but is not used by this rack, so it leaves it alone", unused[0]))
	case len(unused) > 1:
		lines = append(lines, fmt.Sprintf("%s use email validation but are not used by this rack, so it leaves them alone", strings.Join(unused, ", ")))
	}

	for _, r := range d.Records {
		if r.Check != "ok" {
			return append(lines, "When every record shows ok: convox certs migrate --wait")
		}
	}

	return lines
}

func certsDnsNotes(d *structs.CertificateDns) []string {
	lines := []string{}

	switch n := len(d.Preparing); {
	case n == 1:
		lines = append(lines, "1 certificate is still being prepared, run again in a few minutes")
	case n > 1:
		lines = append(lines, fmt.Sprintf("%d certificates are still being prepared, run again in a few minutes", n))
	}

	for _, e := range d.Errors {
		lines = append(lines, fmt.Sprintf("%s: %s", e.Id, e.Error))
	}

	return lines
}

func certsZoneFile(c *stdcli.Context, d *structs.CertificateDns) error {
	lines := []string{}
	zone := ""

	for i, r := range d.Records {
		if i == 0 || r.Zone != zone {
			zone = r.Zone
			lines = append(lines, fmt.Sprintf("; zone %s", zone))
		}

		lines = append(lines, fmt.Sprintf("%s 300 IN CNAME %s", r.Name, r.Value))
	}

	if err := certsWrite(c.Writer().Stdout, lines...); err != nil {
		return err
	}

	return certsWrite(c.Writer().Stderr, certsDnsNotes(d)...)
}

func certsGenerateRecords(rack sdk.Interface, c *stdcli.Context, id string) error {
	d, err := rack.CertificateDns()
	if err != nil || d == nil {
		return nil
	}

	t := c.Table("ZONE", "NAME", "TYPE", "VALUE", "CHECK")

	for _, r := range d.Records {
		if slices.Contains(r.Certificates, id) {
			t.AddRow(r.Zone, r.Name, r.Type, r.Value, r.Check)
		}
	}

	if len(t.Rows) == 0 {
		return certsWrite(c.Writer().Stdout, "Run convox certs dns in a minute to see the records to add")
	}

	if err := t.Print(); err != nil {
		return err
	}

	return certsWrite(c.Writer().Stdout, fmt.Sprintf("Deploy the domain once convox certs shows %s as dns", id))
}

func certsMigrationTable(c *stdcli.Context, ms structs.CertificateMigrations) error {
	t := c.Table("CERTIFICATE", "DOMAIN", "RESULT")

	for _, m := range ms {
		t.AddRow(m.Id, m.Domain, certsMigrationResult(m))
	}

	return t.Print()
}

func certsMigrationResult(m structs.CertificateMigration) string {
	switch m.Result {
	case "waiting":
		return "waiting for its DNS record (run convox certs dns)"
	case "error":
		return fmt.Sprintf("error: %s", m.Detail)
	}

	return m.Result
}

func certsMigrateWait(rack sdk.Interface, c *stdcli.Context, ms structs.CertificateMigrations) error {
	tracked := structs.CertificateMigrations{}

	for _, m := range ms {
		switch m.Result {
		case "preparing", "publishing", "switching":
			tracked = append(tracked, m)
		}
	}

	if len(tracked) == 0 {
		return nil
	}

	c.Startf("Waiting for switches to finish")

	tracked, err := certsMigrationPoll(rack, tracked)
	if err != nil {
		return err
	}

	unfinished := []string{}

	for _, m := range tracked {
		if m.Result != "dns" {
			unfinished = append(unfinished, m.Id)
		}
	}

	if len(unfinished) > 0 {
		if err := certsWrite(c.Writer().Stdout, ""); err != nil {
			return err
		}

		if err := certsMigrationTable(c, tracked); err != nil {
			return err
		}

		return fmt.Errorf("certificates did not finish switching: %s", strings.Join(unfinished, ", "))
	}

	if err := c.OK(); err != nil {
		return err
	}

	return certsMigrationTable(c, tracked)
}

func certsMigrationPoll(rack sdk.Interface, tracked structs.CertificateMigrations) (structs.CertificateMigrations, error) {
	deadline := time.Now().Add(CertsWaitTimeout)

	for time.Now().Before(deadline) {
		ids := []string{}

		for _, m := range tracked {
			if !certsMigrationDone(m.Result) {
				ids = append(ids, m.Id)
			}
		}

		if len(ids) == 0 {
			break
		}

		time.Sleep(CertsWaitInterval)

		ms, err := rack.CertificateMigrationList(ids)
		if certsRouteMissing(err) {
			return nil, certsRackError("migrate", err)
		}
		if err != nil {
			continue
		}

		for _, m := range ms {
			for i := range tracked {
				if tracked[i].Id == m.Id {
					tracked[i] = m
				}
			}
		}
	}

	return tracked, nil
}

func certsMigrationDone(result string) bool {
	switch result {
	case "dns", "error", "failed", "waiting":
		return true
	}

	return false
}

func certsRackError(command string, err error) error {
	if certsRouteMissing(err) {
		return fmt.Errorf("convox certs %s requires a rack on the latest release", command)
	}

	return err
}

func certsRouteMissing(err error) bool {
	if err == nil {
		return false
	}

	for _, s := range []string{"response status 404", "response status 405", "404 page not found"} {
		if strings.Contains(err.Error(), s) {
			return true
		}
	}

	return false
}

func certsWrite(w io.Writer, lines ...string) error {
	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	return nil
}
