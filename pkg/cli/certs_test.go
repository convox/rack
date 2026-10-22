package cli_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/convox/rack/pkg/cli"
	mocksdk "github.com/convox/rack/pkg/mock/sdk"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

func fxCertificateStatus(id, domain, status string, owned bool) structs.Certificate {
	c := *fxCertificate()
	c.Id = id
	c.Domain = domain
	c.Owned = owned
	c.Status = status
	return c
}

func fxCertificateDns() *structs.CertificateDns {
	return &structs.CertificateDns{
		Records: []structs.CertificateDnsRecord{
			{
				Zone:         "example.com",
				Name:         "_0123456789abcdef0123456789abcdef.example.com.",
				Type:         "CNAME",
				Value:        "_fedcba9876543210fedcba9876543210.abcdefghij.acm-validations.aws.",
				Check:        "missing",
				Route53:      true,
				Certificates: []string{"acm-0a1b2c3d4e5f"},
			},
			{
				Zone:         "example.org",
				Name:         "_1123456789abcdef0123456789abcdef.shop.example.org.",
				Type:         "CNAME",
				Value:        "_eedcba9876543210fedcba9876543210.bcdefghijk.acm-validations.aws.",
				Check:        "ok",
				Certificates: []string{"acm-1b2c3d4e5f6a"},
			},
		},
	}
}

func testCertsWait(t *testing.T, interval, timeout time.Duration) {
	i, o := cli.CertsWaitInterval, cli.CertsWaitTimeout
	cli.CertsWaitInterval, cli.CertsWaitTimeout = interval, timeout
	t.Cleanup(func() { cli.CertsWaitInterval, cli.CertsWaitTimeout = i, o })
}

func TestCerts(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateList").Return(structs.Certificates{
			fxCertificateStatus("cert1", "example.org", "dns", true),
			fxCertificateStatus("cert2", "example.com", "pending", true),
			fxCertificateStatus("cert3", "example.net", "email", false),
			fxCertificateStatus("cert4", "*.rack.convox.site", "email", false),
		}, nil)

		res, err := testExecute(e, "certs", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ID     DOMAIN              EXPIRES          STATUS",
			"cert1  example.org         2 days from now  dns",
			"cert2  example.com         2 days from now  pending",
			"cert3  example.net         2 days from now  email",
			"cert4  *.rack.convox.site  2 days from now  email",
		})
	})
}

func TestCertsEmail(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateList").Return(structs.Certificates{
			fxCertificateStatus("cert1", "example.org", "email", true),
			fxCertificateStatus("cert2", "example.com", "email", false),
		}, nil)

		res, err := testExecute(e, "certs", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ID     DOMAIN       EXPIRES          STATUS",
			"cert1  example.org  2 days from now  email",
			"cert2  example.com  2 days from now  email",
			"",
			"1 certificate uses email validation, which stops renewing on 2027-09-30: run convox certs dns",
		})
	})
}

func TestCertsEmailConvoxSite(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateList").Return(structs.Certificates{
			fxCertificateStatus("cert1", "*.Rack.us-east-1.CONVOX.SITE", "email", true),
			fxCertificateStatus("cert2", "example.com", "dns", true),
		}, nil)

		res, err := testExecute(e, "certs", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ID     DOMAIN                        EXPIRES          STATUS",
			"cert1  *.Rack.us-east-1.CONVOX.SITE  2 days from now  email",
			"cert2  example.com                   2 days from now  dns",
			"",
			"1 convox.site certificate is still on email validation: run convox certs migrate to see why",
		})
	})
}

func TestCertsEmailBoth(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateList").Return(structs.Certificates{
			fxCertificateStatus("acm-89ea927329d7", "*.myrack-router-123456789.us-east-1.convox.site", "email", true),
			fxCertificateStatus("acm-79ea927329d7", "web.myrack-router-123456789.us-east-1.convox.site", "email", true),
			fxCertificateStatus("acm-0a1b2c3d4e5f", "example.com", "email", true),
			fxCertificateStatus("acm-1b2c3d4e5f6a", "shop.example.org", "email", true),
			fxCertificateStatus("acm-2b2c3d4e5f6a", "other.example.org", "email", false),
			fxCertificateStatus("acm-3b2c3d4e5f6a", "app.example.org", "migrating", true),
		}, nil)

		res, err := testExecute(e, "certs", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ID                DOMAIN                                             EXPIRES          STATUS",
			"acm-89ea927329d7  *.myrack-router-123456789.us-east-1.convox.site    2 days from now  email",
			"acm-79ea927329d7  web.myrack-router-123456789.us-east-1.convox.site  2 days from now  email",
			"acm-0a1b2c3d4e5f  example.com                                        2 days from now  email",
			"acm-1b2c3d4e5f6a  shop.example.org                                   2 days from now  email",
			"acm-2b2c3d4e5f6a  other.example.org                                  2 days from now  email",
			"acm-3b2c3d4e5f6a  app.example.org                                    2 days from now  migrating",
			"",
			"2 certificates use email validation, which stops renewing on 2027-09-30: run convox certs dns",
			"2 convox.site certificates are still on email validation: run convox certs migrate to see why",
		})
	})
}

func TestCertsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateList").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "certs", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestCertsDelete(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateDelete", "cert1").Return(nil)

		res, err := testExecute(e, "certs delete cert1", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Deleting certificate cert1... OK"})
	})
}

func TestCertsDeleteError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateDelete", "cert1").Return(fmt.Errorf("err1"))

		res, err := testExecute(e, "certs delete cert1", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Deleting certificate cert1... "})
	})
}

func TestCertsGenerate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateGenerate", []string{"test.example.org", "other.example.org"}, structs.CertificateGenerateOptions{Validation: options.String("dns")}).Return(fxCertificate(), nil)
		i.On("CertificateDns").Return(&structs.CertificateDns{
			Records: []structs.CertificateDnsRecord{
				{Zone: "example.org", Name: "_a.test.example.org.", Type: "CNAME", Value: "_b.acm-validations.aws.", Check: "missing", Certificates: []string{"cert1"}},
				{Zone: "example.net", Name: "_c.example.net.", Type: "CNAME", Value: "_d.acm-validations.aws.", Check: "ok", Certificates: []string{"cert2"}},
				{Zone: "example.org", Name: "_e.other.example.org.", Type: "CNAME", Value: "_f.acm-validations.aws.", Check: "ok", Certificates: []string{"cert0", "cert1"}},
			},
		}, nil)

		res, err := testExecute(e, "certs generate test.example.org other.example.org", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Generating certificate... OK, cert1",
			"ZONE         NAME                   TYPE   VALUE                    CHECK",
			"example.org  _a.test.example.org.   CNAME  _b.acm-validations.aws.  missing",
			"example.org  _e.other.example.org.  CNAME  _f.acm-validations.aws.  ok",
			"Deploy the domain once convox certs shows cert1 as dns",
		})
	})
}

func TestCertsGenerateNoRecords(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateGenerate", []string{"test.example.org"}, structs.CertificateGenerateOptions{Validation: options.String("dns")}).Return(fxCertificate(), nil)
		i.On("CertificateDns").Return(&structs.CertificateDns{
			Records: []structs.CertificateDnsRecord{
				{Zone: "example.net", Name: "_c.example.net.", Type: "CNAME", Value: "_d.acm-validations.aws.", Check: "ok", Certificates: []string{"cert2"}},
			},
			Preparing: []string{"cert1"},
		}, nil)

		res, err := testExecute(e, "certs generate test.example.org", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"Generating certificate... OK, cert1",
			"Run convox certs dns in a minute to see the records to add",
		})
	})
}

func TestCertsGenerateDnsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateGenerate", []string{"test.example.org"}, structs.CertificateGenerateOptions{Validation: options.String("dns")}).Return(fxCertificate(), nil)
		i.On("CertificateDns").Return(nil, fmt.Errorf("response status 404"))

		res, err := testExecute(e, "certs generate test.example.org", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Generating certificate... OK, cert1"})
	})
}

func TestCertsGenerateId(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateGenerate", []string{"test.example.org"}, structs.CertificateGenerateOptions{Validation: options.String("dns")}).Return(fxCertificate(), nil)
		i.On("CertificateDns").Return(&structs.CertificateDns{
			Records: []structs.CertificateDnsRecord{
				{Zone: "example.org", Name: "_a.test.example.org.", Type: "CNAME", Value: "_b.acm-validations.aws.", Check: "missing", Certificates: []string{"cert1"}},
			},
		}, nil)

		res, err := testExecute(e, "certs generate test.example.org --id", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{
			"Generating certificate... OK, cert1",
			"ZONE         NAME                  TYPE   VALUE                    CHECK",
			"example.org  _a.test.example.org.  CNAME  _b.acm-validations.aws.  missing",
			"Deploy the domain once convox certs shows cert1 as dns",
		})
		res.RequireStdout(t, []string{"cert1"})
	})
}

func TestCertsGenerateIdNoRecords(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateGenerate", []string{"test.example.org"}, structs.CertificateGenerateOptions{Validation: options.String("dns")}).Return(fxCertificate(), nil)
		i.On("CertificateDns").Return(&structs.CertificateDns{}, nil)

		res, err := testExecute(e, "certs generate test.example.org --id", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{
			"Generating certificate... OK, cert1",
			"Run convox certs dns in a minute to see the records to add",
		})
		res.RequireStdout(t, []string{"cert1"})
	})
}

func TestCertsGenerateIdDnsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateGenerate", []string{"test.example.org"}, structs.CertificateGenerateOptions{Validation: options.String("dns")}).Return(fxCertificate(), nil)
		i.On("CertificateDns").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "certs generate test.example.org --id", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{"Generating certificate... OK, cert1"})
		res.RequireStdout(t, []string{"cert1"})
	})
}

func TestCertsGenerateError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateGenerate", []string{"test.example.org", "other.example.org"}, structs.CertificateGenerateOptions{Validation: options.String("dns")}).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "certs generate test.example.org other.example.org", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Generating certificate... "})
	})
}

func TestCertsImport(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.CertificateCreateOptions{Chain: options.String("chain\n")}
		i.On("CertificateCreate", "cert\n", "key\n", opts).Return(fxCertificate(), nil)

		res, err := testExecute(e, "certs import testdata/cert.pem testdata/key.pem --chain testdata/chain.pem", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Importing certificate... OK, cert1"})
	})
}

func TestCertsImportError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystem(), nil)
		opts := structs.CertificateCreateOptions{Chain: options.String("chain\n")}
		i.On("CertificateCreate", "cert\n", "key\n", opts).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "certs import testdata/cert.pem testdata/key.pem --chain testdata/chain.pem", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{"Importing certificate... "})
	})
}

func TestCertsImportClassic(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("SystemGet").Return(fxSystemClassic(), nil)
		opts := structs.CertificateCreateOptions{Chain: options.String("chain\n")}
		i.On("CertificateCreateClassic", "cert\n", "key\n", opts).Return(fxCertificate(), nil)

		res, err := testExecute(e, "certs import testdata/cert.pem testdata/key.pem --chain testdata/chain.pem", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"Importing certificate... OK, cert1"})
	})
}

func TestCertsDns(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateDns").Return(fxCertificateDns(), nil)
		i.On("CertificateList").Return(structs.Certificates{
			fxCertificateStatus("acm-0a1b2c3d4e5f", "example.com", "email", true),
			fxCertificateStatus("acm-1b2c3d4e5f6a", "shop.example.org", "email", true),
		}, nil)

		res, err := testExecute(e, "certs dns", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ZONE         NAME                                                 TYPE   VALUE                                                              CHECK    CERTIFICATES",
			"example.com  _0123456789abcdef0123456789abcdef.example.com.       CNAME  _fedcba9876543210fedcba9876543210.abcdefghij.acm-validations.aws.  missing  acm-0a1b2c3d4e5f",
			"example.org  _1123456789abcdef0123456789abcdef.shop.example.org.  CNAME  _eedcba9876543210fedcba9876543210.bcdefghijk.acm-validations.aws.  ok       acm-1b2c3d4e5f6a",
			"",
			"2 records: 1 ok, 1 missing",
			"example.com is a Route 53 zone in this AWS account. To have the rack add its records: convox rack params set Route53CertificateValidation=Yes",
			"When every record shows ok: convox certs migrate --wait",
		})
	})
}

func TestCertsDnsLines(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateDns").Return(&structs.CertificateDns{
			Errors: []structs.CertificateDnsError{
				{Id: "cert5", Error: "lookup failed"},
				{Id: "cert6", Error: "certificate not found"},
			},
			Preparing: []string{"cert7", "cert8"},
			Records: []structs.CertificateDnsRecord{
				{Zone: "example.com", Name: "_a.example.com.", Type: "CNAME", Value: "_b.acm-validations.aws.", Check: "wrong", Route53: true, Certificates: []string{"cert1", "cert2"}},
				{Zone: "example.com", Name: "_c.www.example.com.", Type: "CNAME", Value: "_d.acm-validations.aws.", Check: "ok", Route53: true, Certificates: []string{"cert1"}},
				{Zone: "example.net", Name: "_e.example.net.", Type: "CNAME", Value: "_f.acm-validations.aws.", Check: "unknown", Certificates: []string{"cert3"}},
				{Zone: "example.org", Name: "_g.example.org.", Type: "CNAME", Value: "_h.acm-validations.aws.", Check: "unknown", Route53: true, Certificates: []string{"cert4"}},
			},
		}, nil)
		i.On("CertificateList").Return(structs.Certificates{
			fxCertificateStatus("cert1", "example.com", "email", true),
			fxCertificateStatus("cert9", "other.example.com", "email", false),
			fxCertificateStatus("cert10", "*.other.example.net", "email", false),
			fxCertificateStatus("cert11", "*.rack.convox.site", "email", false),
			fxCertificateStatus("cert12", "app.example.net", "dns", false),
		}, nil)

		res, err := testExecute(e, "certs dns", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ZONE         NAME                 TYPE   VALUE                    CHECK    CERTIFICATES",
			"example.com  _a.example.com.      CNAME  _b.acm-validations.aws.  wrong    cert1,cert2",
			"example.com  _c.www.example.com.  CNAME  _d.acm-validations.aws.  ok       cert1",
			"example.net  _e.example.net.      CNAME  _f.acm-validations.aws.  unknown  cert3",
			"example.org  _g.example.org.      CNAME  _h.acm-validations.aws.  unknown  cert4",
			"",
			"4 records: 1 ok, 1 wrong, 2 unknown",
			"example.com is a Route 53 zone in this AWS account. To have the rack add its records: convox rack params set Route53CertificateValidation=Yes",
			"example.org is a Route 53 zone in this AWS account. To have the rack add its records: convox rack params set Route53CertificateValidation=Yes",
			"2 certificates are still being prepared, run again in a few minutes",
			"cert5: lookup failed",
			"cert6: certificate not found",
			"cert9, cert10 use email validation but are not used by this rack, so it leaves them alone",
			"When every record shows ok: convox certs migrate --wait",
		})
	})
}

func TestCertsDnsSingular(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateDns").Return(&structs.CertificateDns{
			Preparing: []string{"cert7"},
			Records: []structs.CertificateDnsRecord{
				{Zone: "example.com", Name: "_a.example.com.", Type: "CNAME", Value: "_b.acm-validations.aws.", Check: "ok", Certificates: []string{"cert1"}},
			},
		}, nil)
		i.On("CertificateList").Return(structs.Certificates{
			fxCertificateStatus("cert1", "example.com", "dns", true),
			fxCertificateStatus("cert9", "other.example.com", "email", false),
		}, nil)

		res, err := testExecute(e, "certs dns", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"ZONE         NAME             TYPE   VALUE                    CHECK  CERTIFICATES",
			"example.com  _a.example.com.  CNAME  _b.acm-validations.aws.  ok     cert1",
			"",
			"1 record: 1 ok",
			"1 certificate is still being prepared, run again in a few minutes",
			"cert9 uses email validation but is not used by this rack, so it leaves it alone",
		})
	})
}

func TestCertsDnsNone(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateDns").Return(&structs.CertificateDns{}, nil)
		i.On("CertificateList").Return(structs.Certificates{
			fxCertificateStatus("cert1", "example.com", "dns", true),
		}, nil)

		res, err := testExecute(e, "certs dns", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{"0 records"})
	})
}

func TestCertsDnsZoneFile(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		d := fxCertificateDns()
		d.Records = append(d.Records, structs.CertificateDnsRecord{
			Zone:         "example.org",
			Name:         "_2123456789abcdef0123456789abcdef.example.org.",
			Type:         "CNAME",
			Value:        "_ddcba9876543210fedcba9876543210.cdefghijkl.acm-validations.aws.",
			Check:        "missing",
			Certificates: []string{"acm-2b2c3d4e5f6a"},
		})
		d.Preparing = []string{"acm-3b2c3d4e5f6a"}
		d.Errors = []structs.CertificateDnsError{{Id: "acm-4b2c3d4e5f6a", Error: "lookup failed"}}
		i.On("CertificateDns").Return(d, nil)

		res, err := testExecute(e, "certs dns --zone-file", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{
			"1 certificate is still being prepared, run again in a few minutes",
			"acm-4b2c3d4e5f6a: lookup failed",
		})
		res.RequireStdout(t, []string{
			"; zone example.com",
			"_0123456789abcdef0123456789abcdef.example.com. 300 IN CNAME _fedcba9876543210fedcba9876543210.abcdefghij.acm-validations.aws.",
			"; zone example.org",
			"_1123456789abcdef0123456789abcdef.shop.example.org. 300 IN CNAME _eedcba9876543210fedcba9876543210.bcdefghijk.acm-validations.aws.",
			"_2123456789abcdef0123456789abcdef.example.org. 300 IN CNAME _ddcba9876543210fedcba9876543210.cdefghijkl.acm-validations.aws.",
		})
	})
}

func TestCertsDnsZoneFileNone(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateDns").Return(&structs.CertificateDns{}, nil)

		res, err := testExecute(e, "certs dns --zone-file", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{""})
	})
}

func TestCertsDnsOldRack(t *testing.T) {
	for _, msg := range []string{"response status 404", "response status 405", "404 page not found"} {
		testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
			i.On("CertificateDns").Return(nil, fmt.Errorf("%s", msg))

			res, err := testExecute(e, "certs dns", nil)
			require.NoError(t, err)
			require.Equal(t, 1, res.Code)
			res.RequireStderr(t, []string{"ERROR: convox certs dns requires a rack on the latest release"})
			res.RequireStdout(t, []string{""})
		})
	}
}

func TestCertsDnsError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateDns").Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "certs dns", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}

func TestCertsMigrate(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateMigrate", []string(nil)).Return(structs.CertificateMigrations{
			{Id: "cert1", Domain: "example.com", Result: "switching"},
			{Id: "cert2", Domain: "example.net", Result: "dns"},
			{Id: "cert3", Domain: "example.org", Result: "waiting"},
			{Id: "cert4", Domain: "rack.convox.site", Result: "publishing"},
			{Id: "cert5", Domain: "app.example.com", Result: "preparing"},
			{Id: "cert6", Domain: "app.example.net", Result: "pending"},
			{Id: "cert7", Domain: "app.example.org", Result: "failed"},
			{Id: "cert8", Domain: "web.example.com", Result: "imported"},
			{Id: "cert9", Domain: "web.example.net", Result: "error", Detail: "registrar 503: registrar unavailable"},
		}, nil)

		res, err := testExecute(e, "certs migrate", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"CERTIFICATE  DOMAIN            RESULT",
			"cert1        example.com       switching",
			"cert2        example.net       dns",
			"cert3        example.org       waiting for its DNS record (run convox certs dns)",
			"cert4        rack.convox.site  publishing",
			"cert5        app.example.com   preparing",
			"cert6        app.example.net   pending",
			"cert7        app.example.org   failed",
			"cert8        web.example.com   imported",
			"cert9        web.example.net   error: registrar 503: registrar unavailable",
		})
	})
}

func TestCertsMigrateIds(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateMigrate", []string{"cert1", "cert2"}).Return(structs.CertificateMigrations{
			{Id: "cert1", Domain: "example.com", Result: "dns"},
			{Id: "cert2", Result: "error", Detail: "not used by this rack"},
		}, nil)

		res, err := testExecute(e, "certs migrate cert1 cert2", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"CERTIFICATE  DOMAIN       RESULT",
			"cert1        example.com  dns",
			"cert2                     error: not used by this rack",
		})
	})
}

func TestCertsMigrateWait(t *testing.T) {
	testCertsWait(t, time.Millisecond, time.Minute)

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateMigrate", []string(nil)).Return(structs.CertificateMigrations{
			{Id: "acm-0a1b2c3d4e5f", Domain: "example.com", Result: "waiting"},
			{Id: "acm-1b2c3d4e5f6a", Domain: "shop.example.org", Result: "switching"},
		}, nil)
		i.On("CertificateMigrationList", []string{"acm-1b2c3d4e5f6a"}).Return(structs.CertificateMigrations{
			{Id: "acm-1b2c3d4e5f6a", Domain: "shop.example.org", Result: "switching"},
		}, nil).Once()
		i.On("CertificateMigrationList", []string{"acm-1b2c3d4e5f6a"}).Return(structs.CertificateMigrations{
			{Id: "acm-1b2c3d4e5f6a", Domain: "shop.example.org", Result: "dns"},
		}, nil).Once()

		res, err := testExecute(e, "certs migrate --wait", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"CERTIFICATE       DOMAIN            RESULT",
			"acm-0a1b2c3d4e5f  example.com       waiting for its DNS record (run convox certs dns)",
			"acm-1b2c3d4e5f6a  shop.example.org  switching",
			"Waiting for switches to finish... OK",
			"CERTIFICATE       DOMAIN            RESULT",
			"acm-1b2c3d4e5f6a  shop.example.org  dns",
		})
	})
}

func TestCertsMigrateWaitPublishing(t *testing.T) {
	testCertsWait(t, time.Millisecond, time.Minute)

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateMigrate", []string{"cert1", "cert2", "cert3"}).Return(structs.CertificateMigrations{
			{Id: "cert1", Domain: "rack.convox.site", Result: "publishing"},
			{Id: "cert2", Domain: "example.com", Result: "preparing"},
			{Id: "cert3", Domain: "example.org", Result: "dns"},
		}, nil)
		i.On("CertificateMigrationList", []string{"cert1", "cert2"}).Return(structs.CertificateMigrations{
			{Id: "cert1", Domain: "rack.convox.site", Result: "dns"},
			{Id: "cert2", Domain: "example.com", Result: "switching"},
		}, nil).Once()
		i.On("CertificateMigrationList", []string{"cert2"}).Return(structs.CertificateMigrations{
			{Id: "cert2", Domain: "example.com", Result: "dns"},
		}, nil).Once()

		res, err := testExecute(e, "certs migrate cert1 cert2 cert3 --wait", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"CERTIFICATE  DOMAIN            RESULT",
			"cert1        rack.convox.site  publishing",
			"cert2        example.com       preparing",
			"cert3        example.org       dns",
			"Waiting for switches to finish... OK",
			"CERTIFICATE  DOMAIN            RESULT",
			"cert1        rack.convox.site  dns",
			"cert2        example.com       dns",
		})
	})
}

func TestCertsMigrateWaitError(t *testing.T) {
	testCertsWait(t, time.Millisecond, time.Minute)

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateMigrate", []string(nil)).Return(structs.CertificateMigrations{
			{Id: "cert1", Domain: "rack.convox.site", Result: "publishing"},
			{Id: "cert2", Domain: "example.com", Result: "switching"},
		}, nil)
		i.On("CertificateMigrationList", []string{"cert1", "cert2"}).Return(structs.CertificateMigrations{
			{Id: "cert1", Domain: "rack.convox.site", Result: "error", Detail: "registrar 503: registrar unavailable"},
			{Id: "cert2", Domain: "example.com", Result: "dns"},
		}, nil).Once()

		res, err := testExecute(e, "certs migrate --wait", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: certificates did not finish switching: cert1"})
		res.RequireStdout(t, []string{
			"CERTIFICATE  DOMAIN            RESULT",
			"cert1        rack.convox.site  publishing",
			"cert2        example.com       switching",
			"Waiting for switches to finish... ",
			"CERTIFICATE  DOMAIN            RESULT",
			"cert1        rack.convox.site  error: registrar 503: registrar unavailable",
			"cert2        example.com       dns",
		})
	})
}

func TestCertsMigrateWaitRetry(t *testing.T) {
	testCertsWait(t, time.Millisecond, time.Minute)

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateMigrate", []string(nil)).Return(structs.CertificateMigrations{
			{Id: "cert1", Domain: "example.com", Result: "switching"},
		}, nil)
		i.On("CertificateMigrationList", []string{"cert1"}).Return(nil, fmt.Errorf("err1")).Once()
		i.On("CertificateMigrationList", []string{"cert1"}).Return(structs.CertificateMigrations{
			{Id: "cert1", Domain: "example.com", Result: "dns"},
		}, nil).Once()

		res, err := testExecute(e, "certs migrate --wait", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"CERTIFICATE  DOMAIN       RESULT",
			"cert1        example.com  switching",
			"Waiting for switches to finish... OK",
			"CERTIFICATE  DOMAIN       RESULT",
			"cert1        example.com  dns",
		})
	})
}

func TestCertsMigrateWaitTimeout(t *testing.T) {
	testCertsWait(t, time.Millisecond, 20*time.Millisecond)

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateMigrate", []string(nil)).Return(structs.CertificateMigrations{
			{Id: "cert1", Domain: "example.com", Result: "switching"},
			{Id: "cert2", Domain: "example.org", Result: "switching"},
		}, nil)
		i.On("CertificateMigrationList", []string{"cert1", "cert2"}).Return(structs.CertificateMigrations{
			{Id: "cert1", Domain: "example.com", Result: "dns"},
			{Id: "cert2", Domain: "example.org", Result: "switching"},
		}, nil).Once()
		i.On("CertificateMigrationList", []string{"cert2"}).Return(structs.CertificateMigrations{
			{Id: "cert2", Domain: "example.org", Result: "pending"},
		}, nil)

		res, err := testExecute(e, "certs migrate --wait", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: certificates did not finish switching: cert2"})
		res.RequireStdout(t, []string{
			"CERTIFICATE  DOMAIN       RESULT",
			"cert1        example.com  switching",
			"cert2        example.org  switching",
			"Waiting for switches to finish... ",
			"CERTIFICATE  DOMAIN       RESULT",
			"cert1        example.com  dns",
			"cert2        example.org  pending",
		})
	})
}

func TestCertsMigrateWaitNothing(t *testing.T) {
	testCertsWait(t, time.Millisecond, time.Minute)

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateMigrate", []string(nil)).Return(structs.CertificateMigrations{
			{Id: "cert1", Domain: "example.com", Result: "waiting"},
			{Id: "cert2", Domain: "example.org", Result: "dns"},
		}, nil)

		res, err := testExecute(e, "certs migrate --wait", nil)
		require.NoError(t, err)
		require.Equal(t, 0, res.Code)
		res.RequireStderr(t, []string{""})
		res.RequireStdout(t, []string{
			"CERTIFICATE  DOMAIN       RESULT",
			"cert1        example.com  waiting for its DNS record (run convox certs dns)",
			"cert2        example.org  dns",
		})
	})
}

func TestCertsMigrateOldRack(t *testing.T) {
	for _, msg := range []string{"response status 404", "response status 405", "404 page not found"} {
		testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
			i.On("CertificateMigrate", []string(nil)).Return(nil, fmt.Errorf("%s", msg))

			res, err := testExecute(e, "certs migrate", nil)
			require.NoError(t, err)
			require.Equal(t, 1, res.Code)
			res.RequireStderr(t, []string{"ERROR: convox certs migrate requires a rack on the latest release"})
			res.RequireStdout(t, []string{""})
		})
	}
}

func TestCertsMigrateWaitOldRack(t *testing.T) {
	testCertsWait(t, time.Millisecond, time.Minute)

	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateMigrate", []string(nil)).Return(structs.CertificateMigrations{
			{Id: "cert1", Domain: "example.com", Result: "switching"},
		}, nil)
		i.On("CertificateMigrationList", []string{"cert1"}).Return(nil, fmt.Errorf("response status 405")).Once()

		res, err := testExecute(e, "certs migrate --wait", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: convox certs migrate requires a rack on the latest release"})
		res.RequireStdout(t, []string{
			"CERTIFICATE  DOMAIN       RESULT",
			"cert1        example.com  switching",
			"Waiting for switches to finish... ",
		})
	})
}

func TestCertsMigrateError(t *testing.T) {
	testClient(t, func(e *cli.Engine, i *mocksdk.Interface) {
		i.On("CertificateMigrate", []string(nil)).Return(nil, fmt.Errorf("err1"))

		res, err := testExecute(e, "certs migrate", nil)
		require.NoError(t, err)
		require.Equal(t, 1, res.Code)
		res.RequireStderr(t, []string{"ERROR: err1"})
		res.RequireStdout(t, []string{""})
	})
}
