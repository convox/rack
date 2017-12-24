package server

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/convox/logger"
	"github.com/convox/nlogger"
	"github.com/convox/rack/api/controllers"
	"github.com/convox/rack/helpers"
	"github.com/convox/rack/provider"
	"github.com/convox/rack/structs"
)

func Listen(name string, addr string) error {
	p := provider.FromName(name)

	if err := p.Initialize(structs.ProviderOptions{}); err != nil {
		return err
	}

	controllers.Provider = p

	n := negroni.New()

	n.Use(negroni.HandlerFunc(recovery))
	n.Use(negroni.HandlerFunc(development))
	n.Use(nlogger.New("ns=api.web", nil))

	n.UseHandler(controllers.NewRouter())

	cert, err := generateSelfSignedCertificate("convox")
	if err != nil {
		return err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	ln, err := tls.Listen("tcp", addr, config)
	if err != nil {
		return err
	}

	return http.Serve(ln, n)
}

func development(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	if os.Getenv("DEVELOPMENT") == "true" {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		rw.Header().Set("Access-Control-Allow-Headers", "*")
		rw.Header().Set("Access-Control-Allow-Methods", "*")
	}

	next(rw, r)
}

func recovery(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	defer recoverWith(func(err error) {
		log := logger.New("ns=kernel").At("panic")
		helpers.Error(log, err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	})

	next(rw, r)
}

func recoverWith(f func(err error)) {
	if r := recover(); r != nil {
		// coerce r to error type
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}

		f(err)
	}
}

func generateSelfSignedCertificate(host string) (tls.Certificate, error) {
	rkey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
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

	return cert, nil
}
