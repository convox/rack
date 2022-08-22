package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"time"
)

func HandleSelfSignedCertificate(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	return GeneratedSelfSignedCertsForDocker(req)
}

func GeneratedSelfSignedCertsForDocker(req Request) (string, map[string]string, error) {
	if req.RequestType == "Create" {
		req.PhysicalResourceId = "cert"
	} else if req.RequestType == "Delete" {
		return req.RequestId, map[string]string{}, nil
	}

	validFor := 3 * 365 * 24 * time.Hour
	rsaBits := 2048

	capriv, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return req.PhysicalResourceId, nil, err
	}

	notBefore := time.Now().Add(-1 * time.Hour)
	notAfter := notBefore.Add(validFor)

	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Convox",
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &capriv.PublicKey, capriv)
	if err != nil {
		return req.PhysicalResourceId, nil, err
	}

	caCertificate, err := x509.ParseCertificate(caBytes)
	if err != nil {
		return req.PhysicalResourceId, nil, err
	}

	caPemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caBytes})

	cakeyBytes := x509.MarshalPKCS1PrivateKey(capriv)
	if err != nil {
		return req.PhysicalResourceId, nil, err
	}
	cakeyPemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: cakeyBytes})

	serverCert := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "Convox",
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		BasicConstraintsValid: true,
	}

	serverpriv, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return req.PhysicalResourceId, nil, err
	}

	serverBytes, err := x509.CreateCertificate(rand.Reader, serverCert, caCertificate, &serverpriv.PublicKey, capriv)
	if err != nil {
		return req.PhysicalResourceId, nil, err
	}

	certPemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverBytes})

	keyBytes := x509.MarshalPKCS1PrivateKey(serverpriv)
	if err != nil {
		return req.PhysicalResourceId, nil, err
	}
	keyPemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})

	return req.PhysicalResourceId, map[string]string{
		"CACert": base64.StdEncoding.EncodeToString(caPemBytes),
		"CAKey":  base64.StdEncoding.EncodeToString(cakeyPemBytes),
		"Cert":   base64.StdEncoding.EncodeToString(certPemBytes),
		"Key":    base64.StdEncoding.EncodeToString(keyPemBytes),
	}, nil
}
