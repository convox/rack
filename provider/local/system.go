package local

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os/exec"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) SystemInstall(w io.Writer, opts structs.SystemInstallOptions) (string, error) {
	if err := checkKubectl(); err != nil {
		return "", err
	}

	if err := checkPermissions(); err != nil {
		return "", err
	}

	name := helpers.DefaultString(opts.Name, "convox")
	version := helpers.DefaultString(opts.Version, "dev")
	url := fmt.Sprintf("https://rack.%s", name)

	fmt.Fprintf(w, "Installing rack (%s)... ", version)

	if err := removeOriginalRack(name); err != nil {
		return "", err
	}

	if _, err := p.Provider.SystemInstall(w, opts); err != nil {
		return "", err
	}

	p.Rack = name

	if err := p.systemUpdate(version); err != nil {
		return "", err
	}

	if err := p.generateCACertificate(name); err != nil {
		return "", err
	}

	if err := dnsInstall(name); err != nil {
		return "", err
	}

	fmt.Fprintf(w, "OK\n")

	fmt.Fprintf(w, "Waiting for rack... ")

	if err := endpointWait(url); err != nil {
		return "", err
	}

	fmt.Fprintf(w, "OK\n")

	return url, nil
}

func (p *Provider) SystemUninstall(name string, w io.Writer, opts structs.SystemUninstallOptions) error {
	if err := checkKubectl(); err != nil {
		return err
	}

	if err := checkPermissions(); err != nil {
		return err
	}

	fmt.Fprintf(w, "Uninstalling rack... ")

	if err := removeOriginalRack(name); err != nil {
		return err
	}

	if err := exec.Command("kubectl", "delete", "ns", "-l", fmt.Sprintf("rack=%s", name)).Run(); err != nil {
		return err
	}

	if err := dnsUninstall(name); err != nil {
		return err
	}

	fmt.Fprintf(w, "OK\n")

	return nil
}

func (p *Provider) SystemUpdate(opts structs.SystemUpdateOptions) error {
	if err := p.Provider.SystemUpdate(opts); err != nil {
		return err
	}

	if err := p.systemUpdate(helpers.DefaultString(opts.Version, p.Version)); err != nil {
		return err
	}

	return nil
}

func (p *Provider) generateCACertificate(name string) error {
	if err := exec.Command("kubectl", "get", "secret", "ca", "-n", "convox-system").Run(); err == nil {
		return nil
	}

	rkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
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
		return err
	}

	pub := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})
	key := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rkey)})

	params := map[string]interface{}{
		"Public":  base64.StdEncoding.EncodeToString(pub),
		"Private": base64.StdEncoding.EncodeToString(key),
	}

	if _, err := p.ApplyTemplate("ca", "system=convox,type=ca", params); err != nil {
		return err
	}

	if err := trustCertificate(name, pub); err != nil {
		return err
	}

	return nil
}

func (p *Provider) systemUpdate(version string) error {
	log := p.logger.At("systemUpdate").Namespace("rack=%s version=%s", p.Rack, version)

	params := map[string]interface{}{
		"DnsPort": dnsPort(),
		"Rack":    p.Rack,
		"Version": version,
	}

	if _, err := p.ApplyTemplate("config", "system=convox,type=config", params); err != nil {
		return log.Error(err)
	}

	if _, err := p.ApplyTemplate("system", "system=convox,type=system", params); err != nil {
		return log.Error(err)
	}

	return log.Success()
}

func checkKubectl() error {
	ch := make(chan error, 1)

	go func() { ch <- exec.Command("kubectl", "version").Run() }()
	go time.AfterFunc(3*time.Second, func() { ch <- fmt.Errorf("timeout") })

	if err := <-ch; err != nil {
		return fmt.Errorf("kubernetes not running or kubectl not configured, try `kubectl version`")
	}

	return nil
}

func endpointWait(url string) error {
	tick := time.Tick(2 * time.Second)
	timeout := time.After(5 * time.Minute)

	ht := *(http.DefaultTransport.(*http.Transport))
	ht.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	hc := &http.Client{Transport: &ht}

	for {
		select {
		case <-tick:
			_, err := hc.Get(url)
			if err == nil {
				return nil
			}
		case <-timeout:
			return fmt.Errorf("timeout")
		}
	}
}
