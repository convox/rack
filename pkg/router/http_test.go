package router_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/convox/rack/pkg/router"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestNoHost(t *testing.T) {
	r := testRouter{}

	testHTTP(t, r, func(h *router.HTTP) {
		res, err := testRequest(h, "GET", "test.convox", nil)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, 502, res.StatusCode)

		data, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, []byte("no route\n"), data)
	})
}

func TestRequestValid(t *testing.T) {
	r := testRouter{}

	testHTTP(t, r, func(h *router.HTTP) {
		port, err := h.Port()
		require.NoError(t, err)

		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NotEmpty(t, r.Header.Get("X-Forwarded-For"))
			require.Equal(t, port, r.Header.Get("X-Forwarded-Port"))
			require.Equal(t, "https", r.Header.Get("X-Forwarded-Proto"))
			fmt.Fprintf(w, "valid")
		}))

		r["test.convox"] = s.URL

		res, err := testRequest(h, "GET", "test.convox", nil)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, 200, res.StatusCode)

		data, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, []byte("valid"), data)
	})
}

func TestRequestRedirect(t *testing.T) {
	r := testRouter{}

	testHTTP(t, r, func(h *router.HTTP) {
		port, err := h.Port()
		require.NoError(t, err)

		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/":
				require.NotEmpty(t, r.Header.Get("X-Forwarded-For"))
				require.Equal(t, port, r.Header.Get("X-Forwarded-Port"))
				require.Equal(t, "https", r.Header.Get("X-Forwarded-Proto"))
				http.Redirect(w, r, "/redirect", 301)
			case "/redirect":
				require.NotEmpty(t, r.Header.Get("X-Forwarded-For"))
				require.Equal(t, port, r.Header.Get("X-Forwarded-Port"))
				require.Equal(t, "https", r.Header.Get("X-Forwarded-Proto"))
				fmt.Fprintf(w, "valid")
			default:
				http.Error(w, "invalid path", 500)
			}
		}))

		r["test.convox"] = s.URL

		res, err := testRequest(h, "GET", "test.convox", nil)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, 200, res.StatusCode)

		data, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, []byte("valid"), data)
	})
}

func TestRequestWebsocket(t *testing.T) {
	r := testRouter{}

	testHTTP(t, r, func(h *router.HTTP) {
		port, err := h.Port()
		require.NoError(t, err)

		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.NotEmpty(t, r.Header.Get("X-Forwarded-For"))
			require.Equal(t, port, r.Header.Get("X-Forwarded-Port"))
			require.Equal(t, "https", r.Header.Get("X-Forwarded-Proto"))

			u := websocket.Upgrader{}

			ws, err := u.Upgrade(w, r, nil)
			require.NoError(t, err)

			wt, data, err := ws.ReadMessage()
			require.NoError(t, err)
			require.Equal(t, websocket.TextMessage, wt)
			require.Equal(t, []byte("input"), data)

			err = ws.WriteMessage(websocket.TextMessage, []byte("output"))
			require.NoError(t, err)
		}))

		r["test.convox"] = s.URL

		ws, err := testWebsocket(h, "test.convox", "/socket")
		require.NoError(t, err)
		defer ws.Close()

		err = ws.WriteMessage(websocket.TextMessage, []byte("input"))
		require.NoError(t, err)

		wt, data, err := ws.ReadMessage()
		require.Equal(t, websocket.TextMessage, wt)
		require.Equal(t, []byte("output"), data)
	})
}

func testHTTP(t *testing.T, r testRouter, fn func(h *router.HTTP)) {
	h, err := router.NewHTTP("https", 0, r)
	require.NoError(t, err)
	defer h.Close()
	go h.Serve()
	fn(h)
}

func testRequest(h *router.HTTP, method, host string, body io.Reader) (*http.Response, error) {
	port, err := h.Port()
	if err != nil {
		return nil, err
	}

	c := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         host,
			},
		},
	}

	req, err := http.NewRequest(method, fmt.Sprintf("https://localhost:%s", port), body)
	if err != nil {
		return nil, err
	}

	req.Host = host

	return c.Do(req)
}

func testWebsocket(h *router.HTTP, host, path string) (*websocket.Conn, error) {
	port, err := h.Port()
	if err != nil {
		return nil, err
	}

	d := websocket.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         host,
		},
	}

	hs := http.Header{}
	hs.Set("Host", host)

	c, _, err := d.Dial(fmt.Sprintf("wss://localhost:%s%s", port, path), hs)
	if err != nil {
		return nil, err
	}

	return c, nil
}

type testRouter map[string]string

func (r testRouter) Certificate(host string) (*tls.Certificate, error) {
	rkey, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		return nil, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

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

	return &cert, nil
}

func (r testRouter) Route(host string) (string, error) {
	target, ok := r[host]
	if !ok {
		return "", fmt.Errorf("no route")
	}

	return target, nil
}
