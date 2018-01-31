package router

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"
)

func caCertificate() (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair("/Users/Shared/convox/ca.crt", "/Users/Shared/convox/ca.key")
	if err != nil {
		return generateCACertificate()
	}

	cert, err = tls.LoadX509KeyPair("/etc/convox/ca.crt", "/etc/convox/ca.key")
	if err != nil {
		return generateCACertificate()
	}

	return cert, nil
}

func generateCACertificate() (tls.Certificate, error) {
	rkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:         true,
		DNSNames:     []string{"ca.convox"},
		SerialNumber: serial,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		Subject: pkix.Name{
			CommonName:   "ca.convox",
			Organization: []string{"convox"},
		},
	}

	data, err := x509.CreateCertificate(rand.Reader, &template, &template, &rkey.PublicKey, rkey)
	if err != nil {
		return tls.Certificate{}, err
	}

	pub := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})
	key := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rkey)})

	cert, err := tls.X509KeyPair(pub, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	if err := os.MkdirAll("/etc/convox", 0755); err != nil {
		return tls.Certificate{}, err
	}

	if err := ioutil.WriteFile("/etc/convox/ca.crt", pub, 0644); err != nil {
		return tls.Certificate{}, err
	}

	if err := ioutil.WriteFile("/etc/convox/ca.key", key, 0600); err != nil {
		return tls.Certificate{}, err
	}

	return cert, nil
}

func (r *Router) generateCertificate(host string) (tls.Certificate, error) {
	rkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}

	cpub, err := x509.ParseCertificate(r.ca.Certificate[0])
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   host,
			Organization: []string{"convox"},
		},
		Issuer:                cpub.Subject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{host, fmt.Sprintf("*.%s", host)},
	}

	data, err := x509.CreateCertificate(rand.Reader, &template, cpub, &rkey.PublicKey, r.ca.PrivateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	pub := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})
	key := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rkey)})

	return tls.X509KeyPair(pub, key)
}
