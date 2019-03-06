package router

import (
	crand "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"
	"time"

	"golang.org/x/crypto/acme/autocert"
	am "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	idleCheck   = 1 * time.Minute
	idleTimeout = 60 * time.Minute
)

var (
	targetParser = regexp.MustCompile(`^([^.]+)\.([^.]+)\.svc\.cluster\.local$`)
)

type Router struct {
	Cluster kubernetes.Interface
	DNS     Server
	HTTP    Server
	HTTPS   Server
	IP      string

	backend Backend
	certs   sync.Map
}

type Server interface {
	ListenAndServe() error
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func New() (*Router, error) {
	r := &Router{
		certs: sync.Map{},
	}

	switch os.Getenv("BACKEND") {
	case "dynamodb":
		r.backend = NewBackendDynamo(os.Getenv("ROUTER_ROUTES"))
	default:
		r.backend = NewBackendMemory()
	}

	c, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	kc, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}

	r.Cluster = kc

	dns, err := NewDNS(r)
	if err != nil {
		return nil, err
	}

	r.DNS = dns

	if err := r.setupHTTP(); err != nil {
		return nil, err
	}

	ic, err := NewIngressController(r)
	if err != nil {
		return nil, err
	}

	go ic.Run()

	s, err := kc.CoreV1().Services("convox-system").Get("router", am.GetOptions{})
	if err != nil {
		return nil, err
	}

	if len(s.Status.LoadBalancer.Ingress) > 0 && s.Status.LoadBalancer.Ingress[0].Hostname == "localhost" {
		r.IP = "127.0.0.1"
	} else {
		r.IP = s.Spec.ClusterIP
	}

	return r, nil
}

func (r *Router) Serve() error {
	ch := make(chan error, 1)

	go serve(ch, r.DNS)
	go serve(ch, r.HTTP)
	go serve(ch, r.HTTPS)

	go r.idleTicker()

	return <-ch
}

func (r *Router) RequestBegin(host string) error {
	idle, err := r.backend.IdleGet(host)
	if err != nil {
		return fmt.Errorf("could not fetch idle status: %s", err)
	}

	if idle {
		if err := r.HostUnidle(host); err != nil {
			return fmt.Errorf("could not unidle: %s", err)
		}
	}

	return r.backend.RequestBegin(host)
}

func (r *Router) RequestEnd(host string) error {
	return r.backend.RequestEnd(host)
}

func (r *Router) Route(host string) (string, error) {
	ts, err := r.TargetList(host)
	if err != nil {
		return "", fmt.Errorf("no backends available")
	}

	if len(ts) < 1 {
		return "", fmt.Errorf("no backends available")
	}

	return ts[rand.Intn(len(ts))], nil
}

func (r *Router) TargetAdd(host, target string) error {
	fmt.Printf("ns=convox.router at=target.add host=%q target=%q\n", host, target)

	idle, err := r.HostIdleStatus(host)
	if err != nil {
		return err
	}

	if err := r.backend.IdleSet(host, idle); err != nil {
		return err
	}

	return r.backend.TargetAdd(host, target)
}

func (r *Router) TargetList(host string) ([]string, error) {
	return r.backend.TargetList(host)
}

func (r *Router) TargetRemove(host, target string) error {
	fmt.Printf("ns=convox.router at=target.delete host=%q target=%q\n", host, target)

	return r.backend.TargetRemove(host, target)
}

func (r *Router) generateCertificateAutocert(m *autocert.Manager) func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		if hello.ServerName == "" {
			return generateSelfSignedCertificate("")
		}

		return m.GetCertificate(hello)
	}
}

func (r *Router) generateCertificateCA(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	host := hello.ServerName

	v, ok := r.certs.Load(host)
	if ok {
		if c, ok := v.(tls.Certificate); ok {
			return &c, nil
		}
	}

	c, err := r.Certificate(host)
	if err != nil {
		return nil, err
	}

	r.certs.Store(host, *c)

	return c, nil
}

func (r *Router) setupHTTP() error {
	if os.Getenv("AUTOCERT") == "true" {
		return r.setupHTTPAutocert()
	}

	ln, err := tls.Listen("tcp", fmt.Sprintf(":443"), &tls.Config{
		GetCertificate: r.generateCertificateCA,
	})
	if err != nil {
		return err
	}

	https, err := NewHTTP(ln, r)
	if err != nil {
		return err
	}

	r.HTTPS = https

	r.HTTP = &http.Server{Addr: ":80", Handler: redirectHTTPS(https.ServeHTTP)}

	return nil
}

func (r *Router) setupHTTPAutocert() error {
	m := &autocert.Manager{
		Cache:  NewCacheDynamo(os.Getenv("ROUTER_CACHE")),
		Prompt: autocert.AcceptTOS,
	}

	ln, err := tls.Listen("tcp", fmt.Sprintf(":443"), &tls.Config{
		GetCertificate: r.generateCertificateAutocert(m),
	})
	if err != nil {
		return err
	}

	https, err := NewHTTP(ln, r)
	if err != nil {
		return err
	}

	r.HTTPS = https

	r.HTTP = &http.Server{Addr: ":80", Handler: m.HTTPHandler(redirectHTTPS(https.ServeHTTP))}

	return nil
}

func generateSelfSignedCertificate(host string) (*tls.Certificate, error) {
	rkey, err := rsa.GenerateKey(crand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serial, err := crand.Int(crand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   host,
			Organization: []string{"convox"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{host},
	}

	data, err := x509.CreateCertificate(crand.Reader, &template, &template, &rkey.PublicKey, rkey)

	if err != nil {
		return nil, err
	}

	pub := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data})
	key := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rkey)})

	cert, err := tls.X509KeyPair(pub, key)
	if err != nil {
		return nil, err
	}

	return &cert, nil
}

func parseTarget(target string) (string, string, bool) {
	u, err := url.Parse(target)
	if err != nil {
		return "", "", false
	}

	if m := targetParser.FindStringSubmatch(u.Hostname()); len(m) == 3 {
		return m[1], m[2], true
	}

	return "", "", false
}

func redirectHTTPS(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			fn(w, r)
			return
		}

		target := url.URL{Scheme: "https", Host: r.Host, Path: r.URL.Path, RawQuery: r.URL.RawQuery}

		http.Redirect(w, r, target.String(), http.StatusMovedPermanently)
	}
}

func serve(ch chan error, s Server) {
	ch <- s.ListenAndServe()
}
