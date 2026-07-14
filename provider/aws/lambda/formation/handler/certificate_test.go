package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ssm"
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

func makeCertB64(t *testing.T, validFor time.Duration) string {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %s", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-2 * time.Hour),
		NotAfter:     time.Now().Add(validFor),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create certificate: %s", err)
	}
	return base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func TestShouldRegenerateCert(t *testing.T) {
	notPEM := base64.StdEncoding.EncodeToString([]byte("hello"))
	pemNotCert := base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("not a certificate")}))

	cases := []struct {
		name string
		cert string
		want bool
	}{
		{"valid long-lived", makeCertB64(t, 100*365*24*time.Hour), false},
		{"expiring within two months", makeCertB64(t, 30*24*time.Hour), true},
		{"already expired", makeCertB64(t, -1*time.Hour), true},
		{"not base64", "!!! not base64 !!!", true},
		{"base64 but not PEM", notPEM, true},
		{"PEM but not a certificate", pemNotCert, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldRegenerateCert(c.cert, time.Now()); got != c.want {
				t.Errorf("shouldRegenerateCert(%q) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

func TestIsParameterNotFound(t *testing.T) {
	if !isParameterNotFound(awserr.New(ssm.ErrCodeParameterNotFound, "not found", nil)) {
		t.Error("ParameterNotFound should be detected")
	}
	if !isParameterNotFound(awserr.NewRequestFailure(awserr.New(ssm.ErrCodeParameterNotFound, "not found", nil), 400, "req-id")) {
		t.Error("RequestFailure-wrapped ParameterNotFound should be detected")
	}
	if !isParameterNotFound(&ssm.ParameterNotFound{}) {
		t.Error("typed *ssm.ParameterNotFound should be detected")
	}
	if isParameterNotFound(awserr.New("ThrottlingException", "slow down", nil)) {
		t.Error("a transient aws error must not be treated as not-found")
	}
	if isParameterNotFound(errors.New("network blip")) {
		t.Error("a non-aws error must not be treated as not-found")
	}
	if isParameterNotFound(nil) {
		t.Error("nil error must not be treated as not-found")
	}
}
