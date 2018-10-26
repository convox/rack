// +build !windows

package token

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/convox/hid"
	"github.com/convox/u2f/u2fhid"
	"github.com/convox/u2f/u2ftoken"
)

type authenticationKey struct {
	AppId     string `json:"appId"`
	KeyHandle string `json:"keyHandle"`
	Version   string `json:"version"`
}

type authenticationRequest struct {
	AppId          string `json:"appId"`
	Challenge      string `json:"challenge"`
	RegisteredKeys []authenticationKey
}

type authenticationResponse struct {
	ClientData    string `json:"clientData"`
	KeyHandle     string `json:"keyHandle"`
	SignatureData string `json:"signatureData"`
}

type tokenResponse struct {
	Error    error
	Response []byte
}

func decodeBase64(s string) ([]byte, error) {
	for i := 0; i < len(s)%4; i++ {
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

func encodeBase64(buf []byte) string {
	s := base64.URLEncoding.EncodeToString(buf)
	return strings.TrimRight(s, "=")
}

func Authenticate(req []byte) ([]byte, error) {
	var areq authenticationRequest

	if err := json.Unmarshal(req, &areq); err != nil {
		return nil, err
	}

	ds, err := u2fhid.Devices()
	if err != nil {
		if err != nil {
			return nil, err
		}
	}

	ch := make(chan tokenResponse)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, d := range ds {
		go authenticateDevice(ctx, d, areq, ch)
	}

	for range ds {
		res := <-ch

		if res.Error != nil {
			return nil, res.Error
		}

		if res.Response != nil {
			return res.Response, nil
		}
	}

	return nil, fmt.Errorf("no valid tokens found")
}

func authenticateDevice(ctx context.Context, d *hid.DeviceInfo, req authenticationRequest, rch chan tokenResponse) {
	ud, err := u2fhid.Open(d)
	if err != nil {
		rch <- tokenResponse{Error: err}
		return
	}

	cd := []byte(fmt.Sprintf(`{"challenge":"%s","origin":"%s"}`, req.Challenge, req.AppId))
	ch := sha256.Sum256(cd)

	t := u2ftoken.NewToken(ud)

	for _, k := range req.RegisteredKeys {
		app := sha256.Sum256([]byte(k.AppId))

		key, err := decodeBase64(k.KeyHandle)
		if err != nil {
			rch <- tokenResponse{Error: err}
			return
		}

		treq := u2ftoken.AuthenticateRequest{
			Application: app[:],
			Challenge:   ch[:],
			KeyHandle:   key,
		}

		if err := t.CheckAuthenticate(treq); err != nil {
			continue
		}

		for {
			tres, err := t.Authenticate(treq)
			if err == u2ftoken.ErrPresenceRequired {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if err != nil {
				continue
			}

			ares := authenticationResponse{
				ClientData:    encodeBase64(cd),
				KeyHandle:     k.KeyHandle,
				SignatureData: encodeBase64(tres.RawResponse),
			}

			data, err := json.Marshal(ares)
			if err != nil {
				rch <- tokenResponse{Error: err}
				return
			}

			rch <- tokenResponse{Response: data}
			return
		}
	}

	rch <- tokenResponse{}
}
