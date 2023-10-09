package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
)

const (
	certArnSeperator = "$#$"
)

func HandleSelfSignedCertificate(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	_, ok := req.ResourceProperties["Rack"].(string)
	if !ok {
		return "", nil, fmt.Errorf("Rack property is required")
	}

	switch req.RequestType {
	case "Create":
		return CreateSelfSignedCertsForDocker(req)
	case "Update":
		return UpdateSelfSignedCertsForDocker(req)
	case "Delete":
		return DeleteSelfSignedCertsForDocker(req)
	}
	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func HandleSelfSignedCertificateGetter(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	_, ok := req.ResourceProperties["Rack"].(string)
	if !ok {
		return "", nil, fmt.Errorf("Rack property is required")
	}

	switch req.RequestType {
	case "Create", "Update", "Delete":
		return GetCertificate(req)
	}
	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func CreateSelfSignedCertsForDocker(req Request) (string, map[string]string, error) {
	certsMap, err := generateSelfSignedCertsForDocker()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate cert: %s", err)
	}

	rackKey := rackHash(req.ResourceProperties["Rack"].(string))

	ssmClient := SSM(req)

	_, err = ssmClient.PutParameter(&ssm.PutParameterInput{
		Name:      aws.String(caCertParameterName(rackKey)),
		Overwrite: aws.Bool(true),
		Value:     aws.String(certsMap["CACert"]),
		Type:      aws.String(ssm.ParameterTypeString),
	})
	if err != nil {
		return "", nil, err
	}

	_, err = ssmClient.PutParameter(&ssm.PutParameterInput{
		Name:      aws.String(caKeyParameterName(rackKey)),
		Overwrite: aws.Bool(true),
		Value:     aws.String(certsMap["CAKey"]),
		Type:      aws.String(ssm.ParameterTypeString),
	})
	if err != nil {
		return "", nil, err
	}

	_, err = ssmClient.PutParameter(&ssm.PutParameterInput{
		Name:      aws.String(certParameterName(rackKey)),
		Overwrite: aws.Bool(true),
		Value:     aws.String(certsMap["Cert"]),
		Type:      aws.String(ssm.ParameterTypeString),
	})
	if err != nil {
		return "", nil, err
	}

	_, err = ssmClient.PutParameter(&ssm.PutParameterInput{
		Name:      aws.String(keyParameterName(rackKey)),
		Overwrite: aws.Bool(true),
		Value:     aws.String(certsMap["Key"]),
		Type:      aws.String(ssm.ParameterTypeString),
	})
	if err != nil {
		return "", nil, err
	}

	_, err = ssmClient.PutParameter(&ssm.PutParameterInput{
		Name:      aws.String(versionParameterName(rackKey)),
		Overwrite: aws.Bool(true),
		Value:     aws.String(req.ResourceProperties["Version"].(string)),
		Type:      aws.String(ssm.ParameterTypeString),
	})
	if err != nil {
		return "", nil, err
	}

	return rackKey, map[string]string{
		"CACertSSMKey": caCertParameterName(rackKey),
		"CAKeySSMKey":  caKeyParameterName(rackKey),
		"CertSSMKey":   certParameterName(rackKey),
		"KeySSMKey":    keyParameterName(rackKey),
	}, nil
}

func DeleteSelfSignedCertsForDocker(req Request) (string, map[string]string, error) {
	rackKey := rackHash(req.ResourceProperties["Rack"].(string))
	ssmClient := SSM(req)

	ssmClient.DeleteParameter(&ssm.DeleteParameterInput{
		Name: aws.String(caCertParameterName(rackKey)),
	})
	ssmClient.DeleteParameter(&ssm.DeleteParameterInput{
		Name: aws.String(caKeyParameterName(rackKey)),
	})
	ssmClient.DeleteParameter(&ssm.DeleteParameterInput{
		Name: aws.String(certParameterName(rackKey)),
	})
	ssmClient.DeleteParameter(&ssm.DeleteParameterInput{
		Name: aws.String(keyParameterName(rackKey)),
	})

	return rackKey, map[string]string{
		"CACertSSMKey": caCertParameterName(rackKey),
		"CAKeySSMKey":  caKeyParameterName(rackKey),
		"CertSSMKey":   certParameterName(rackKey),
		"KeySSMKey":    keyParameterName(rackKey),
	}, nil
}

func GetCertificate(req Request) (string, map[string]string, error) {
	rackKey := rackHash(req.ResourceProperties["Rack"].(string))
	key := req.ResourceProperties["ParameterKey"].(string)
	param, err := SSM(req).GetParameter(&ssm.GetParameterInput{
		Name: aws.String(key),
	})
	if err != nil {
		return "", nil, err
	}

	return rackKey, map[string]string{
		"Value": *param.Parameter.Value,
	}, nil
}

func UpdateSelfSignedCertsForDocker(req Request) (string, map[string]string, error) {
	rackKey := rackHash(req.ResourceProperties["Rack"].(string))
	ssmClient := SSM(req)
	param, err := ssmClient.GetParameter(&ssm.GetParameterInput{
		Name: aws.String(versionParameterName(rackKey)),
	})
	if err != nil || param.Parameter == nil || param.Parameter.Value == nil ||
		*param.Parameter.Value != req.ResourceProperties["Version"].(string) {
		return CreateSelfSignedCertsForDocker(req)
	}

	pemBytes, err := base64.StdEncoding.DecodeString(*param.Parameter.Value)
	if err != nil {
		return CreateSelfSignedCertsForDocker(req)
	}

	block, _ := pem.Decode(pemBytes)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return CreateSelfSignedCertsForDocker(req)
	}

	// renew if cert expires in 2 months
	if cert.NotAfter.Before(time.Now().Add(2 * 30 * 24 * time.Hour)) {
		return CreateSelfSignedCertsForDocker(req)
	}

	return rackKey, map[string]string{
		"CACertSSMKey": caCertParameterName(rackKey),
		"CAKeySSMKey":  caKeyParameterName(rackKey),
		"CertSSMKey":   certParameterName(rackKey),
		"KeySSMKey":    keyParameterName(rackKey),
	}, nil
}

func caCertParameterName(rack string) string {
	return fmt.Sprintf("%s-docker-tls-ca-cert", rack)
}

func caKeyParameterName(rack string) string {
	return fmt.Sprintf("%s-docker-tls-ca-key", rack)
}

func certParameterName(rack string) string {
	return fmt.Sprintf("%s-docker-tls-cert", rack)
}

func keyParameterName(rack string) string {
	return fmt.Sprintf("%s-docker-tls-key", rack)
}

func versionParameterName(rack string) string {
	return fmt.Sprintf("%s-version-track", rack)
}

func generateSelfSignedCertsForDocker() (map[string]string, error) {
	validFor := 100 * 365 * 24 * time.Hour
	rsaBits := 2048

	capriv, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	caPemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caBytes})

	caCertificate, err := x509.ParseCertificate(caBytes)
	if err != nil {
		return nil, err
	}

	cakeyBytes := x509.MarshalPKCS1PrivateKey(capriv)
	if err != nil {
		return nil, err
	}

	cakeyPemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: cakeyBytes})

	cert := &x509.Certificate{
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

	certpriv, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, caCertificate, &certpriv.PublicKey, capriv)
	if err != nil {
		return nil, err
	}

	certPemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})

	keyBytes := x509.MarshalPKCS1PrivateKey(certpriv)
	if err != nil {
		return nil, err
	}

	keyPemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})

	return map[string]string{
		"CACert": base64.StdEncoding.EncodeToString(caPemBytes),
		"CAKey":  base64.StdEncoding.EncodeToString(cakeyPemBytes),
		"Cert":   base64.StdEncoding.EncodeToString(certPemBytes),
		"Key":    base64.StdEncoding.EncodeToString(keyPemBytes),
	}, nil
}

func rackHash(rack string) string {
	return hex.EncodeToString([]byte(rack))
}
