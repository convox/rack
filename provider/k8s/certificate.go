package k8s

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
	ac "k8s.io/api/core/v1"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (p *Provider) CertificateApply(app, service string, port int, id string) error {
	return fmt.Errorf("unimplemented")
}

func (p *Provider) CertificateCreate(pub, key string, opts structs.CertificateCreateOptions) (*structs.Certificate, error) {
	s, err := p.Cluster.CoreV1().Secrets(p.Rack).Create(&ac.Secret{
		ObjectMeta: am.ObjectMeta{
			GenerateName: "cert-",
			Labels: map[string]string{
				"system": "convox",
				"rack":   p.Rack,
				"type":   "certificate",
			},
		},
		Data: map[string][]byte{
			"tls.crt": []byte(base64.StdEncoding.EncodeToString([]byte(pub + helpers.DefaultString(opts.Chain, "")))),
			"tls.key": []byte(base64.StdEncoding.EncodeToString([]byte(key))),
		},
		Type: "kubernetes.io/tls",
	})
	if err != nil {
		return nil, err
	}

	c, err := p.certificateFromSecret(s)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (p *Provider) CertificateDelete(id string) error {
	if err := p.Cluster.CoreV1().Secrets(p.Rack).Delete(id, nil); err != nil {
		return err
	}

	return nil
}

func (p *Provider) CertificateGenerate(domains []string) (*structs.Certificate, error) {
	// pub, key, err := p.generateCertificate(domains)
	// if err != nil {
	//   return nil, err
	// }

	// return p.CertificateCreate(string(pub), string(key), structs.CertificateCreateOptions{})
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) CertificateList() (structs.Certificates, error) {
	ss, err := p.Cluster.CoreV1().Secrets(p.Rack).List(am.ListOptions{
		FieldSelector: "type=kubernetes.io/tls",
		LabelSelector: fmt.Sprintf("system=convox,rack=%s,type=certificate", p.Rack),
	})
	if err != nil {
		return nil, err
	}

	cs := structs.Certificates{}

	for _, s := range ss.Items {
		c, err := p.certificateFromSecret(&s)
		if err != nil {
			return nil, err
		}

		cs = append(cs, *c)
	}

	return cs, nil
}

// func (p *Provider) caCertificate() (*tls.Certificate, error) {
//   c, err := p.Cluster.CoreV1().Secrets(p.Rack).Get("ca", am.GetOptions{})
//   if ae.IsNotFound(err) {
//     return p.generateCACertificate()
//   }
//   if err != nil {
//     return nil, err
//   }

//   crt, err := base64.StdEncoding.DecodeString(string(c.Data["tls.crt"]))
//   if err != nil {
//     return nil, err
//   }

//   key, err := base64.StdEncoding.DecodeString(string(c.Data["tls.key"]))
//   if err != nil {
//     return nil, err
//   }

//   ca, err := tls.X509KeyPair(crt, key)
//   if err != nil {
//     return nil, err
//   }

//   return &ca, nil
// }

func (p *Provider) certificateFromSecret(s *ac.Secret) (*structs.Certificate, error) {
	cert, ok := s.Data["tls.crt"]
	if !ok {
		return nil, fmt.Errorf("invalid certificate: %s", s.ObjectMeta.Name)
	}

	data, err := base64.StdEncoding.DecodeString(string(cert))
	if err != nil {
		return nil, err
	}

	pb, _ := pem.Decode(data)

	if pb.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("invalid certificate: %s", s.ObjectMeta.Name)
	}

	cs, err := x509.ParseCertificates(pb.Bytes)
	if err != nil {
		return nil, err
	}

	if len(cs) < 1 {
		return nil, fmt.Errorf("invalid certificate: %s", s.ObjectMeta.Name)
	}

	c := &structs.Certificate{
		Id:         s.ObjectMeta.Name,
		Domain:     cs[0].Subject.CommonName,
		Domains:    cs[0].DNSNames,
		Expiration: cs[0].NotAfter,
	}

	return c, nil
}

func (p *Provider) generateCACertificate() (*tls.Certificate, error) {
	rkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"ca.convox"},
		SerialNumber:          serial,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		Subject: pkix.Name{
			CommonName:   "ca.convox",
			Organization: []string{"convox"},
		},
	}

	data, err := x509.CreateCertificate(rand.Reader, &template, &template, &rkey.PublicKey, rkey)
	if err != nil {
		return nil, err
	}

	pub := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})
	key := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rkey)})

	cert, err := tls.X509KeyPair(pub, key)
	if err != nil {
		return nil, err
	}

	_, err = p.Cluster.CoreV1().Secrets(p.Rack).Create(&ac.Secret{
		ObjectMeta: am.ObjectMeta{
			Name: "ca",
			Labels: map[string]string{
				"system": "convox",
				"rack":   p.Rack,
				"type":   "ca",
			},
		},
		Data: map[string][]byte{
			"tls.crt": []byte(base64.StdEncoding.EncodeToString(pub)),
			"tls.key": []byte(base64.StdEncoding.EncodeToString(key)),
		},
		Type: "kubernetes.io/tls",
	})
	if err != nil {
		return nil, err
	}

	return &cert, nil
}

// func (p *Provider) generateCertificate(domains []string) ([]byte, []byte, error) {
//   if len(domains) < 1 {
//     return nil, nil, fmt.Errorf("must specify at least one domain")
//   }

//   ca, err := p.caCertificate()
//   if err != nil {
//     return nil, nil, err
//   }

//   rkey, err := rsa.GenerateKey(rand.Reader, 2048)
//   if err != nil {
//     return nil, nil, err
//   }

//   serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
//   if err != nil {
//     return nil, nil, err
//   }

//   cpub, err := x509.ParseCertificate(ca.Certificate[0])
//   if err != nil {
//     return nil, nil, err
//   }

//   template := x509.Certificate{
//     SerialNumber: serial,
//     Subject: pkix.Name{
//       CommonName:   domains[0],
//       Organization: []string{"convox"},
//     },
//     Issuer:                cpub.Subject,
//     NotBefore:             time.Now(),
//     NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
//     KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
//     ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
//     BasicConstraintsValid: true,
//   }

//   if len(domains) > 1 {
//     template.DNSNames = domains[1:]
//   }

//   data, err := x509.CreateCertificate(rand.Reader, &template, cpub, &rkey.PublicKey, ca.PrivateKey)
//   if err != nil {
//     return nil, nil, err
//   }

//   pub := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})
//   key := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rkey)})

//   return pub, key, nil
// }
