package cli

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/convox/rack/pkg/token"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
	"github.com/convox/stdsdk"
)

var reSessionAuthentication = regexp.MustCompile(`^Session path="([^"]+)" token="([^"]+)"$`)

type AuthenticationError struct {
	error
}

func (ae AuthenticationError) AuthenticationError() error {
	return ae.error
}

type session struct {
	Id string `json:"id"`
}

func authenticator(c *stdcli.Context) stdsdk.Authenticator {
	return func(cl *stdsdk.Client, res *http.Response) (http.Header, error) {
		m := reSessionAuthentication.FindStringSubmatch(res.Header.Get("WWW-Authenticate"))
		if len(m) < 3 {
			return nil, nil
		}

		body := []byte{}
		headers := map[string]string{}

		if m[2] == "true" {
			ares, err := cl.GetStream(m[1], stdsdk.RequestOptions{})
			if err != nil {
				return nil, err
			}
			defer ares.Body.Close()

			dres, err := io.ReadAll(ares.Body)
			if err != nil {
				return nil, err
			}

			c.Writef("Waiting for security token... ")

			if os.Getenv("CONVOX_WEB_U2F_DISABLE") == "true" {
				data, err := token.Authenticate(dres)
				if err != nil {
					return nil, AuthenticationError{err}
				}
				body = data
			} else {
				data, err := browserAuthenticate(c, cl, dres)
				if err != nil {
					return nil, AuthenticationError{err}
				}
				body = data
			}

			c.Writef("<ok>OK</ok>\n")

			headers["Challenge"] = ares.Header.Get("Challenge")
		}

		var s session

		ro := stdsdk.RequestOptions{
			Body:    bytes.NewReader(body),
			Headers: stdsdk.Headers(headers),
		}

		if err := cl.Post(m[1], ro, &s); err != nil {
			return nil, err
		}

		if s.Id == "" {
			return nil, fmt.Errorf("invalid session")
		}

		if err := c.SettingWriteKey("session", cl.Endpoint.Host, s.Id); err != nil {
			return nil, err
		}

		h := http.Header{}

		h.Set("Session", s.Id)

		return h, nil
	}
}

func currentSession(c *stdcli.Context) sdk.SessionFunc {
	return func(cl *sdk.Client) string {
		sid, _ := c.SettingReadKey("session", cl.Endpoint.Host)
		return sid
	}
}

func browserAuthenticate(c *stdcli.Context, cl *stdsdk.Client, challenge []byte) ([]byte, error) {
	target := url.URL{
		Scheme: cl.Endpoint.Scheme,
		Host:   cl.Endpoint.Host,
		Path:   "/login/u2f",
	}

	dataChan := make(chan []byte, 1)
	errChan := make(chan error, 1)

	addr, srv, err := u2fCallbackServer(dataChan, errChan)
	if err != nil {
		return nil, err
	}
	defer srv.Close()

	q := target.Query()
	q.Add("token", base64.StdEncoding.EncodeToString(challenge))
	q.Add("callback_url", addr)
	target.RawQuery = q.Encode()

	c.Writef("\nOpen this link in your browser to complete authentication:\n%s\n", target.String())
	c.Writef("(on a headless machine, set CONVOX_WEB_U2F_DISABLE=true to use a USB security key instead)\n")

	if err := openBrowser(target.String()); err != nil {
		c.Writef("Could not open a browser automatically; open the link above manually.\n")
	}

	timeout := time.NewTimer(5 * time.Minute)
	defer timeout.Stop()

	select {
	case <-timeout.C:
		return nil, fmt.Errorf("timed out waiting for browser authentication; set CONVOX_WEB_U2F_DISABLE=true to use a USB security key")
	case err := <-errChan:
		return nil, err
	case data := <-dataChan:
		return data, nil
	}
}

func u2fCallbackServer(dataChan chan []byte, errChan chan error) (string, *http.Server, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, fmt.Errorf("failed to listen on port: %s", err)
	}

	addr := fmt.Sprintf("http://127.0.0.1:%d", listener.Addr().(*net.TCPAddr).Port)

	srv := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data := r.URL.Query().Get("data")
			if data == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// query parsing decodes '+' to a space; standard base64 never contains spaces
			data = strings.ReplaceAll(data, " ", "+")

			decoded, err := base64.StdEncoding.DecodeString(data)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "Authentication complete. You may close this window.")

			select {
			case dataChan <- decoded:
			default:
			}
		}),
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			select {
			case errChan <- err:
			default:
			}
		}
	}()

	return addr, srv, nil
}

func openBrowser(uri string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", uri).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", uri).Start()
	case "darwin":
		return exec.Command("open", uri).Start()
	default:
		return fmt.Errorf("unsupported platform")
	}
}
