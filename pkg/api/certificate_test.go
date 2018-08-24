package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/stdsdk"
	"github.com/stretchr/testify/require"
)

var fxCertificate = structs.Certificate{
	Id:         "cert1",
	Domain:     "domain",
	Domains:    []string{"domain1", "domain2"},
	Expiration: time.Now().UTC(),
}

func TestCertificateApply(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"id": "cert1",
			},
		}
		p.On("CertificateApply", "app1", "service1", 5000, "cert1").Return(nil)
		err := c.Put("/apps/app1/ssl/service1/5000", ro, nil)
		require.NoError(t, err)
	})
}

func TestCertificateApplyError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"id": "cert1",
			},
		}
		p.On("CertificateApply", "app1", "service1", 5000, "cert1").Return(fmt.Errorf("err1"))
		err := c.Put("/apps/app1/ssl/service1/5000", ro, nil)
		require.EqualError(t, err, "err1")
	})
}

func TestCertificateCreate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		c1 := fxCertificate
		c2 := structs.Certificate{}
		opts := structs.CertificateCreateOptions{
			Chain: options.String("chain"),
		}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"chain": "chain",
				"pub":   "pub",
				"key":   "key",
			},
		}
		p.On("CertificateCreate", "pub", "key", opts).Return(&c1, nil)
		err := c.Post("/certificates", ro, &c2)
		require.NoError(t, err)
		require.Equal(t, c1, c2)
	})
}

func TestCertificateCreateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var c1 *structs.Certificate
		p.On("CertificateCreate", "", "", structs.CertificateCreateOptions{}).Return(nil, fmt.Errorf("err1"))
		err := c.Post("/certificates", stdsdk.RequestOptions{}, c1)
		require.EqualError(t, err, "err1")
		require.Nil(t, c1)
	})
}

func TestCertificateDelete(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("CertificateDelete", "cert1").Return(nil)
		err := c.Delete("/certificates/cert1", stdsdk.RequestOptions{}, nil)
		require.NoError(t, err)
	})
}

func TestCertificateDeleteError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		p.On("CertificateDelete", "cert1").Return(fmt.Errorf("err1"))
		err := c.Delete("/certificates/cert1", stdsdk.RequestOptions{}, nil)
		require.EqualError(t, err, "err1")
	})
}

func TestCertificateGenerate(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		c1 := fxCertificate
		c2 := structs.Certificate{}
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"domains": "domain1,domain2",
			},
		}
		p.On("CertificateGenerate", []string{"domain1", "domain2"}).Return(&c1, nil)
		err := c.Post("/certificates/generate", ro, &c2)
		require.NoError(t, err)
		require.Equal(t, c1, c2)
	})
}

func TestCertificateGenerateError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var c1 *structs.Certificate
		ro := stdsdk.RequestOptions{
			Params: stdsdk.Params{
				"domains": "domain1,domain2",
			},
		}
		p.On("CertificateGenerate", []string{"domain1", "domain2"}).Return(nil, fmt.Errorf("err1"))
		err := c.Post("/certificates/generate", ro, c1)
		require.EqualError(t, err, "err1")
		require.Nil(t, c1)
	})
}

func TestCertificateList(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		c1 := structs.Certificates{fxCertificate, fxCertificate}
		c2 := structs.Certificates{}
		p.On("CertificateList").Return(c1, nil)
		err := c.Get("/certificates", stdsdk.RequestOptions{}, &c2)
		require.NoError(t, err)
		require.Equal(t, c1, c2)
	})
}

func TestCertificateListError(t *testing.T) {
	testServer(t, func(c *stdsdk.Client, p *structs.MockProvider) {
		var c1 structs.Certificates
		p.On("CertificateList").Return(nil, fmt.Errorf("err1"))
		err := c.Get("/certificates", stdsdk.RequestOptions{}, &c1)
		require.EqualError(t, err, "err1")
		require.Nil(t, c1)
	})
}
