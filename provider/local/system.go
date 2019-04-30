package local

import (
	"bytes"
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
	"os"
	"os/exec"
	"runtime"
	"sort"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/storage"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/provider/k8s"
	"github.com/convox/rack/sdk"
)

var (
	systemTemplates = []string{"registry", "router"}
)

func (p *Provider) SystemHost() string {
	return fmt.Sprintf("rack.%s", p.Rack)
}

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

	p.Rack = name

	fmt.Fprintf(w, "Installing... ")

	if err := removeOriginalRack(name); err != nil {
		return "", err
	}

	if err := networkSetup(); err != nil {
		return "", err
	}

	if err := dnsInstall(name); err != nil {
		return "", err
	}

	if err := p.generateCACertificate(version); err != nil {
		return "", err
	}

	if _, err := p.Provider.SystemInstall(w, opts); err != nil {
		return "", err
	}

	params := map[string]interface{}{
		"DNS":  dnsPort(),
		"Rack": name,
	}

	if err := p.systemUpdate(version, params); err != nil {
		return "", err
	}

	fmt.Fprintf(w, "OK\n")

	fmt.Fprintf(w, "Starting... ")

	if err := endpointWait(url); err != nil {
		return "", err
	}

	fmt.Fprintf(w, "OK\n")

	if err := importOriginalSettings(w, name, url); err != nil {
		return "", err
	}

	return url, nil
}

func (p *Provider) SystemStatus() (string, error) {
	return "running", nil
}

func (p *Provider) SystemTemplate(version string) ([]byte, error) {
	params := map[string]interface{}{
		"Version": version,
	}

	ts := [][]byte{}

	data, err := p.Provider.SystemTemplate(version)
	if err != nil {
		return nil, err
	}

	ts = append(ts, data)

	for _, st := range systemTemplates {
		data, err := p.RenderTemplate(fmt.Sprintf("system/%s", st), params)
		if err != nil {
			return nil, err
		}

		ldata, err := k8s.ApplyLabels(data, "system=convox,provider=local")
		if err != nil {
			return nil, err
		}

		ts = append(ts, ldata)
	}

	return bytes.Join(ts, []byte("---\n")), nil
}

func (p *Provider) SystemUninstall(name string, w io.Writer, opts structs.SystemUninstallOptions) error {
	if err := checkKubectl(); err != nil {
		return err
	}

	if err := checkPermissions(); err != nil {
		return err
	}

	fmt.Fprintf(w, "Uninstalling... ")

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

	version := helpers.DefaultString(opts.Version, p.Version)

	params := map[string]interface{}{
		"DNS":  os.Getenv("DNS"),
		"Rack": p.Rack,
	}

	if err := p.systemUpdate(version, params); err != nil {
		return err
	}

	return nil
}

func (p *Provider) generateCACertificate(version string) error {
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

	data, err = p.RenderTemplate("ca", params)
	if err != nil {
		return err
	}

	if err := p.Apply(p.Rack, "ca", version, data, fmt.Sprintf("system=convox,provider=local,rack=%s", p.Rack), 30); err != nil {
		return err
	}

	if err := trustCertificate(p.Rack, pub); err != nil {
		return err
	}

	return nil
}

func (p *Provider) systemUpdate(version string, params map[string]interface{}) error {
	data, err := p.RenderTemplate("config", params)
	if err != nil {
		return err
	}

	if err := p.Apply(p.Rack, "config", version, data, fmt.Sprintf("system=convox,provider=local,rack=%s", p.Rack), 30); err != nil {
		return err
	}

	host := fmt.Sprintf("rack.%s", p.Rack)
	template := fmt.Sprintf("https://convox.s3.amazonaws.com/release/%s/provider/local/k8s/rack.yml", version)

	data, err = helpers.Get(template)
	if err != nil {
		return err
	}

	tags := map[string]string{
		"DNS":    helpers.CoalesceString(os.Getenv("DNS"), dnsPort()),
		"HOST":   host,
		"RACK":   p.Rack,
		"SOCKET": dockerSocket(),
	}

	for k, v := range tags {
		data = bytes.Replace(data, []byte(fmt.Sprintf("==%s==", k)), []byte(v), -1)
	}

	if err := p.Apply(p.Rack, "system", version, data, fmt.Sprintf("system=convox,provider=local,rack=%s", p.Rack), 300); err != nil {
		return err
	}

	return nil
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
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	timeout := time.After(5 * time.Minute)

	ht := *(http.DefaultTransport.(*http.Transport))
	ht.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	hc := &http.Client{Timeout: 2 * time.Second, Transport: &ht}

	for {
		select {
		case <-tick.C:
			res, err := hc.Get(fmt.Sprintf("%s/apps", url))
			if err == nil && res.StatusCode == 200 {
				return nil
			}
		case <-timeout:
			return fmt.Errorf("timeout")
		}
	}
}

func importOriginalEnvironment(s *storage.Storage, app string) (string, error) {
	ris, err := s.List(fmt.Sprintf("apps/%s/releases", app))
	if err != nil {
		return "", err
	}

	rs := structs.Releases{}

	for _, ri := range ris {
		var r structs.Release

		if err := s.Load(fmt.Sprintf("apps/%s/releases/%s/release.json", app, ri), &r); err != nil {
			return "", err
		}

		rs = append(rs, r)
	}

	sort.Slice(rs, rs.Less)

	if len(rs) < 1 {
		return "", nil
	}

	return rs[0].Env, nil
}

func importOriginalSettings(w io.Writer, name, url string) error {
	db := ""

	switch runtime.GOOS {
	case "darwin":
		db = fmt.Sprintf("/Users/Shared/convox/%s.db", name)
	case "linux":
		db = fmt.Sprintf("/var/convox/%s.db", name)
	default:
		return nil
	}

	if _, err := os.Stat(db); os.IsNotExist(err) {
		return nil
	}

	fmt.Fprintf(w, "Importing original rack settings... ")

	c, err := sdk.New(url)
	if err != nil {
		return err
	}

	s, err := storage.Open(db)
	if err != nil {
		return err
	}
	defer s.Close()

	cas, err := c.AppList()
	if err != nil {
		return err
	}

	cash := map[string]bool{}

	for _, ca := range cas {
		cash[ca.Name] = true
	}

	eas, err := s.List("apps")
	if err != nil {
		return err
	}

	for _, ea := range eas {
		if cash[ea] {
			continue
		}

		env, err := importOriginalEnvironment(s, ea)
		if err != nil {
			return err
		}

		if _, err := c.AppCreate(ea, structs.AppCreateOptions{}); err != nil {
			return err
		}

		if _, err := c.ReleaseCreate(ea, structs.ReleaseCreateOptions{Env: options.String(env)}); err != nil {
			return err
		}
	}

	if err := s.Close(); err != nil {
		return err
	}

	if err := os.Rename(db, fmt.Sprintf("%s.backup", db)); err != nil {
		return err
	}

	fmt.Fprintf(w, "OK\n")

	return nil
}
