package structs

import (
	"strings"
	"time"

	"github.com/gobwas/glob"
)

type Certificate struct {
	Arn        string    `json:"-"`
	Id         string    `json:"id"`
	Domain     string    `json:"domain"`
	Domains    []string  `json:"domains"`
	Expiration time.Time `json:"expiration"`
	Owned      bool      `json:"owned"`
	Status     string    `json:"status"`
}

type Certificates []Certificate

type CertificateDns struct {
	Errors    []CertificateDnsError  `json:"errors"`
	Preparing []string               `json:"preparing"`
	Records   []CertificateDnsRecord `json:"records"`
}

type CertificateDnsError struct {
	Id    string `json:"id"`
	Error string `json:"error"`
}

type CertificateDnsRecord struct {
	Zone         string   `json:"zone"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Value        string   `json:"value"`
	Check        string   `json:"check"`
	Route53      bool     `json:"route53"`
	Certificates []string `json:"certificates"`
}

type CertificateMigration struct {
	Id     string `json:"id"`
	Domain string `json:"domain"`
	Result string `json:"result"`
	Detail string `json:"detail"`
}

type CertificateMigrations []CertificateMigration

type CertificateCreateOptions struct {
	Chain *string `param:"chain"`
}

func (c Certificates) Less(i, j int) bool { return strings.ToUpper(c[i].Id) < strings.ToUpper(c[j].Id) }

func (c *Certificate) Match(domain string) (bool, error) {
	for _, d := range c.Domains {
		g, err := glob.Compile(d, '.')
		if err != nil {
			return false, err
		}

		if g.Match(domain) {
			return true, nil
		}
	}

	return false, nil
}

type TLSPemCertBytes struct {
	CACert []byte
	CAKey  []byte
	Cert   []byte
	Key    []byte
}
