package u2fhost

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	butil "github.com/marshallbrekka/go-u2fhost/bytes"
)

// Authenticates with the device using the AuthenticateRequest,
// returning an AuthenticateResponse.
func (dev *HidDevice) Authenticate(req *AuthenticateRequest) (*AuthenticateResponse, error) {
	clientData, request, err := authenticateRequest(req)
	if err != nil {
		return nil, err
	}

	authModifier := u2fAuthEnforce
	if req.CheckOnly {
		authModifier = u2fAuthCheckOnly
	}

	status, response, err := dev.hidDevice.SendAPDU(u2fCommandAuthenticate, authModifier, 0, request)
	if err != nil {
		return nil, err
	}

	if status == u2fStatusNoError {
		response := authenticateResponse(status, response, clientData, req)

		// Clear out the authenticator data if the original request was not webauthn.
		if !req.WebAuthn {
			response.AuthenticatorData = ""
		}
		return response, nil
	}

	// If we are in webauthn mode, try a backwards compatible mode for u2f
	if req.WebAuthn && status == u2fStatusWrongData {
		u2fReq := *req
		u2fReq.WebAuthn = false
		u2fReq.AppId = "https://" + req.AppId
		clientData, request, err = authenticateRequest(&u2fReq)

		if err != nil {
			return nil, err
		}

		status, response, err = dev.hidDevice.SendAPDU(u2fCommandAuthenticate, authModifier, 0, request)
		if err != nil {
			return nil, err
		}

		if status == u2fStatusNoError {
			return authenticateResponse(status, response, clientData, &u2fReq), nil
		}
	}

	return nil, u2ferror(status)
}

func authenticateResponse(status uint16, response, clientData []byte, req *AuthenticateRequest) *AuthenticateResponse {
	authenticatorData := append(sha256([]byte(req.AppId)), response[0:5]...)
	if req.WebAuthn {
		return &AuthenticateResponse{
			KeyHandle:         req.KeyHandle,
			ClientData:        websafeEncode(clientData),
			SignatureData:     base64.StdEncoding.EncodeToString(response[5:]),
			AuthenticatorData: base64.StdEncoding.EncodeToString(authenticatorData),
		}
	} else {
		return &AuthenticateResponse{
			KeyHandle:         req.KeyHandle,
			ClientData:        websafeEncode(clientData),
			SignatureData:     websafeEncode(response),
			AuthenticatorData: base64.StdEncoding.EncodeToString(authenticatorData),
		}
	}
}

func authenticateRequest(req *AuthenticateRequest) ([]byte, []byte, error) {
	// Get the channel id public key, if any
	cid, err := channelIdPublicKey(req.ChannelIdPublicKey, req.ChannelIdUnused)
	if err != nil {
		return nil, nil, err
	}

	// Construct the client json
	keyHandle, err := websafeDecode(req.KeyHandle)
	if err != nil {
		return []byte{}, []byte{}, fmt.Errorf("base64 key handle: %s", err)
	}

	client := clientData{
		Challenge:          req.Challenge,
		Origin:             req.Facet,
		ChannelIdPublicKey: cid,
	}

	if req.WebAuthn {
		client.Type = "webauthn.get"
	} else {
		client.Typ = "navigator.id.getAssertion"
	}

	clientJson, err := json.Marshal(client)
	if err != nil {
		return nil, nil, fmt.Errorf("Error marshaling clientData to json: %s", err)
	}

	// Pack into byte array
	// https://fidoalliance.org/specs/fido-u2f-v1.0-nfc-bt-amendment-20150514/fido-u2f-raw-message-formats.html#authentication-request-message---u2f_authenticate
	request := butil.Concat(
		sha256(clientJson),
		sha256([]byte(req.AppId)),
		[]byte{byte(len(keyHandle))},
		keyHandle,
	)
	return []byte(clientJson), request, nil
}
