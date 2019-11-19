package cli

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

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

			dres, err := ioutil.ReadAll(ares.Body)
			if err != nil {
				return nil, err
			}

			c.Writef("Waiting for security token... ")

			data, err := token.Authenticate(dres)
			if err != nil {
				return nil, AuthenticationError{err}
			}

			c.Writef("<ok>OK</ok>\n")

			body = data
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
