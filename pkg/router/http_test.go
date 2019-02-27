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
	"strings"
	"testing"
	"time"

	"github.com/convox/rack/pkg/router"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestNoHost(t *testing.T) {
	r := testRouter{}

	testHTTP(t, r, func(h *router.HTTP) {
		res, err := testRequest(h, "GET", "test.convox", nil, nil)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, 502, res.StatusCode)

		data, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, []byte("no route\n"), data)
	})
}

func TestRequest(t *testing.T) {
	r := testRouter{}

	testHTTP(t, r, func(h *router.HTTP) {
		port, err := h.Port()
		require.NoError(t, err)

		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "convox/router-test", r.Header.Get("User-Agent"))
			require.NotEmpty(t, r.Header.Get("X-Forwarded-For"))
			require.Equal(t, port, r.Header.Get("X-Forwarded-Port"))
			require.Equal(t, "https", r.Header.Get("X-Forwarded-Proto"))
			fmt.Fprintf(w, "valid")
		}))

		r["test.convox"] = s.URL

		res, err := testRequest(h, "GET", "test.convox", nil, nil)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, 200, res.StatusCode)

		data, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, []byte("valid"), data)
	})
}

func TestRequestError(t *testing.T) {
	r := testRouter{}

	testHTTP(t, r, func(h *router.HTTP) {
		r["test.convox"] = "://invalid"

		res, err := testRequest(h, "GET", "test.convox", nil, nil)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, 502, res.StatusCode)

		data, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, []byte("invalid target: ://invalid\n"), data)
	})
}

func TestRequestHTTPS(t *testing.T) {
	r := testRouter{}

	testHTTP(t, r, func(h *router.HTTP) {
		port, err := h.Port()
		require.NoError(t, err)

		s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "convox/router-test", r.Header.Get("User-Agent"))
			require.NotEmpty(t, r.Header.Get("X-Forwarded-For"))
			require.Equal(t, port, r.Header.Get("X-Forwarded-Port"))
			require.Equal(t, "https", r.Header.Get("X-Forwarded-Proto"))
			fmt.Fprintf(w, "valid")
		}))

		r["test.convox"] = s.URL

		res, err := testRequest(h, "GET", "test.convox", nil, nil)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, 200, res.StatusCode)

		data, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, []byte("valid"), data)
	})
}

func TestRequestPost(t *testing.T) {
	r := testRouter{}

	testHTTP(t, r, func(h *router.HTTP) {
		port, err := h.Port()
		require.NoError(t, err)

		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "convox/router-test", r.Header.Get("User-Agent"))

			require.NotEmpty(t, r.Header.Get("X-Forwarded-For"))
			require.Equal(t, port, r.Header.Get("X-Forwarded-Port"))
			require.Equal(t, "https", r.Header.Get("X-Forwarded-Proto"))

			require.Equal(t, "7", r.Header.Get("Content-Length"))

			data, err := ioutil.ReadAll(r.Body)
			require.NoError(t, err)
			require.Equal(t, []byte("foo=bar"), data)

			fmt.Fprintf(w, "valid")
		}))

		r["test.convox"] = s.URL

		res, err := testRequest(h, "POST", "test.convox", strings.NewReader("foo=bar"), nil)
		require.NoError(t, err)
		defer res.Body.Close()

		require.Equal(t, 200, res.StatusCode)

		data, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, []byte("valid"), data)
	})
}

func TestRequestExistingForwardHeaders(t *testing.T) {
	r := testRouter{}

	testHTTP(t, r, func(h *router.HTTP) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "convox/router-test", r.Header.Get("User-Agent"))
			require.NotEmpty(t, r.Header.Get("X-Forwarded-For"))
			require.Equal(t, "5000", r.Header.Get("X-Forwarded-Port"))
			require.Equal(t, "foo", r.Header.Get("X-Forwarded-Proto"))
			fmt.Fprintf(w, "valid")
		}))

		r["test.convox"] = s.URL

		hs := http.Header{}

		hs.Set("X-Forwarded-Port", "5000")
		hs.Set("X-Forwarded-Proto", "foo")

		res, err := testRequest(h, "GET", "test.convox", nil, hs)
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
				require.Equal(t, "convox/router-test", r.Header.Get("User-Agent"))
				require.NotEmpty(t, r.Header.Get("X-Forwarded-For"))
				require.Equal(t, port, r.Header.Get("X-Forwarded-Port"))
				require.Equal(t, "https", r.Header.Get("X-Forwarded-Proto"))
				http.Redirect(w, r, "/redirect", 301)
			case "/redirect":
				require.Equal(t, "convox/router-test", r.Header.Get("User-Agent"))
				require.NotEmpty(t, r.Header.Get("X-Forwarded-For"))
				require.Equal(t, port, r.Header.Get("X-Forwarded-Port"))
				require.Equal(t, "https", r.Header.Get("X-Forwarded-Proto"))
				fmt.Fprintf(w, "valid")
			default:
				http.Error(w, "invalid path", 500)
			}
		}))

		r["test.convox"] = s.URL

		res, err := testRequest(h, "GET", "test.convox", nil, nil)
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
			require.Equal(t, "convox/router-test", r.Header.Get("User-Agent"))
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
	h, err := router.NewHTTP(0, r)
	require.NoError(t, err)
	defer h.Close()
	go h.ListenAndServe()
	fn(h)
}

func testRequest(h *router.HTTP, method, host string, body io.Reader, headers http.Header) (*http.Response, error) {
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

	req.Header.Set("User-Agent", "convox/router-test")

	if headers != nil {
		for k, vs := range headers {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}

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
	hs.Set("User-Agent", "convox/router-test")

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
