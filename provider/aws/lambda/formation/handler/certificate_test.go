package handler

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"testing"
	"time"
)

func parseCertParam(t *testing.T, b64pem string) *x509.Certificate {
	t.Helper()
	der, err := base64.StdEncoding.DecodeString(b64pem)
	if err != nil {
		t.Fatalf("base64 decode: %s", err)
	}
	block, _ := pem.Decode(der)
	if block == nil {
		t.Fatal("pem decode returned nil block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse certificate: %s", err)
	}
	return cert
}

func TestGenerateSelfSignedCertsForDockerChainsToCA(t *testing.T) {
	certs, err := generateSelfSignedCertsForDocker()
	if err != nil {
		t.Fatalf("generate: %s", err)
	}

	ca := parseCertParam(t, certs["CACert"])
	leaf := parseCertParam(t, certs["Cert"])

	if !ca.IsCA {
		t.Error("CA cert is not marked IsCA")
	}

	if err := leaf.CheckSignatureFrom(ca); err != nil {
		t.Fatalf("leaf does not chain to CA: %s", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(ca)
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny}}); err != nil {
		t.Fatalf("leaf verify against CA pool: %s", err)
	}

	if got := time.Until(leaf.NotAfter); got < 99*365*24*time.Hour {
		t.Errorf("leaf validity too short: %s remaining", got)
	}
}
