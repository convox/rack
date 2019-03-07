package router_test

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/router"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestHTTPNoHost(t *testing.T) {
	r := testHTTPRouter{}

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

func TestHTTPRequest(t *testing.T) {
	r := testHTTPRouter{}

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

func TestHTTPRequestError(t *testing.T) {
	r := testHTTPRouter{}

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

func TestHTTPRequestHTTPS(t *testing.T) {
	r := testHTTPRouter{}

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

func TestHTTPRequestPost(t *testing.T) {
	r := testHTTPRouter{}

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

func TestHTTPRequestExistingForwardHeaders(t *testing.T) {
	r := testHTTPRouter{}

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

func TestHTTPRequestRedirect(t *testing.T) {
	r := testHTTPRouter{}

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

func TestHTTPRequestWebsocket(t *testing.T) {
	r := testHTTPRouter{}

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

func generateSelfSignedCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return helpers.CertificateSelfSigned(hello.ServerName)
}

func testHTTP(t *testing.T, r testHTTPRouter, fn func(h *router.HTTP)) {
	ln, err := tls.Listen("tcp", "", &tls.Config{
		GetCertificate: generateSelfSignedCertificate,
	})
	require.NoError(t, err)

	h, err := router.NewHTTP(ln, r)
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

type testHTTPRouter map[string]string

func (r testHTTPRouter) RequestBegin(host string) error {
	return nil
}

func (r testHTTPRouter) RequestEnd(host string) error {
	return nil
}

func (r testHTTPRouter) Route(host string) (string, error) {
	target, ok := r[host]
	if !ok {
		return "", fmt.Errorf("no route")
	}

	return target, nil
}
