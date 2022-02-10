// +build !windows

package token

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/convox/go-u2fhost"
)

type clientExtension struct {
	Appid bool `json:"appid"`
}
type authenticationResponse struct {
	AuthenticatorData string `json:"authenticatorData"`
	ClientDataJSON    string `json:"clientDataJSON"`
	Signature         string `json:"signature"`
	UserHandle        string `json:"userHandle"`
}
type webauthnResponse struct {
	ID                     string                 `json:"id"`
	RawID                  string                 `json:"rawId"`
	Type                   string                 `json:"type"`
	ClientExtensionResults clientExtension        `json:"clientExtensionResults"`
	Response               authenticationResponse `json:"response"`
}

type authenticationRequest struct {
	PublicKey struct {
		Challenge        string `json:"challenge"`
		Timeout          int    `json:"timeout"`
		RpID             string `json:"rpId"`
		AllowCredentials []struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		} `json:"allowCredentials"`
		UserVerification string `json:"userVerification"`
	} `json:"publicKey"`
}

type tokenResponse struct {
	Error    error
	Response *u2fhost.AuthenticateResponse
}

func Authenticate(req []byte) ([]byte, error) {
	ds := u2fhost.Devices()

	ch := make(chan tokenResponse)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var areq authenticationRequest
	if err := json.Unmarshal(req, &areq); err != nil {
		return nil, err
	}
	chbase64, err := fromStdToWebEnconding(areq.PublicKey.Challenge)
	if err != nil {
		return nil, err
	}
	areq.PublicKey.Challenge = chbase64

	for _, d := range ds {
		go authenticateWait(ctx, d, areq, ch)
	}

	for range ds {
		res := <-ch

		if res.Error != nil {
			return nil, res.Error
		}

		if res.Response != nil {
			ares := webauthnResponse{
				ID:    res.Response.KeyHandle,
				RawID: res.Response.KeyHandle,
				Type:  "public-key",
				ClientExtensionResults: clientExtension{
					Appid: true,
				},
				Response: authenticationResponse{
					AuthenticatorData: res.Response.AuthenticatorData,
					Signature:         res.Response.SignatureData,
					ClientDataJSON:    res.Response.ClientData,
				},
			}

			data, err := json.Marshal(ares)
			if err != nil {
				return nil, err
			}

			return data, nil
		}
	}

	return nil, fmt.Errorf("no valid tokens found")
}

func authenticateWait(ctx context.Context, d *u2fhost.HidDevice, req authenticationRequest, rch chan tokenResponse) {
	if err := d.Open(); err != nil {
		rch <- tokenResponse{Error: err}
		return
	}
	defer d.Close()

	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()

	ch := make(chan tokenResponse)
	refresh := make(chan bool)

	go authenticateDevice(ctx, d, req, ch, refresh)

	for {
		select {
		case <-timeout.C:
			rch <- tokenResponse{Error: fmt.Errorf("timeout")}
			return
		case <-refresh:
			timeout.Reset(2 * time.Second)
		case res := <-ch:
			rch <- res
			return
		}
	}
}

func authenticateDevice(ctx context.Context, d *u2fhost.HidDevice, req authenticationRequest, ch chan tokenResponse, refresh chan bool) {
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			for _, k := range req.PublicKey.AllowCredentials {
				base64kh, err := fromStdToWebEnconding(k.ID)
				if err != nil {
					ch <- tokenResponse{Error: err}
				}

				origin := fmt.Sprintf("https://%s", req.PublicKey.RpID)
				areq := &u2fhost.AuthenticateRequest{
					AppId:     req.PublicKey.RpID,
					Challenge: req.PublicKey.Challenge,
					Facet:     origin,
					WebAuthn:  true,
					KeyHandle: base64kh,
				}

				refresh <- true

				ares, err := d.Authenticate(areq)
				switch err.(type) {
				case *u2fhost.BadKeyHandleError:
				case *u2fhost.TestOfUserPresenceRequiredError:
				case nil:
					ch <- tokenResponse{Response: ares}
					return
				default:
					ch <- tokenResponse{Error: err}
					return
				}
			}
		}
	}
}

// fromStdToWebEnconding convert the key handle to base64
// then convert it to rawurl enconding, that is expected by the lib
func fromStdToWebEnconding(kh string) (string, error) {
	basehk, err := base64.StdEncoding.DecodeString(kh)
	if err != nil {
		return "", fmt.Errorf("base64 key handle: %s", err)
	}

	return base64.RawURLEncoding.EncodeToString(basehk), nil
}
