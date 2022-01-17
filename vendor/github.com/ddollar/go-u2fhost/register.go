package u2fhost

import (
	"encoding/json"
	"fmt"

	butil "github.com/marshallbrekka/go-u2fhost/bytes"
)

// Registers with the device using the RegisterRequest, returning a RegisterResponse.
func (dev *HidDevice) Register(req *RegisterRequest) (*RegisterResponse, error) {
	clientData, request, err := registerRequest(req)
	if err != nil {
		return nil, err
	}
	var p1 uint8 = 0x03
	var p2 uint8 = 0
	status, response, err := dev.hidDevice.SendAPDU(u2fCommandRegister, p1, p2, request)
	return registerResponse(status, response, clientData, err)
}

func registerRequest(req *RegisterRequest) ([]byte, []byte, error) {
	// Get the channel id public key, if any
	cid, err := channelIdPublicKey(req.ChannelIdPublicKey, req.ChannelIdUnused)
	if err != nil {
		return nil, nil, err
	}

	// Construct the client json
	client := clientData{
		Typ:                "navigator.id.finishEnrollment",
		Challenge:          req.Challenge,
		Origin:             req.Facet,
		ChannelIdPublicKey: cid,
	}
	clientJson, err := json.Marshal(client)
	if err != nil {
		return nil, nil, fmt.Errorf("Error marshaling clientData to json: %s", err)
	}

	// Pack into byte array
	// https://fidoalliance.org/specs/fido-u2f-v1.0-nfc-bt-amendment-20150514/fido-u2f-raw-message-formats.html#registration-request-message---u2f_register
	request := butil.Concat(
		sha256(clientJson),
		sha256([]byte(req.AppId)),
	)
	return []byte(clientJson), request, nil
}

func registerResponse(status uint16, response, clientData []byte, err error) (*RegisterResponse, error) {
	var registerResponse *RegisterResponse
	if err == nil {
		if status == u2fStatusNoError {
			registerResponse = &RegisterResponse{
				RegistrationData: websafeEncode(response),
				ClientData:       websafeEncode(clientData),
			}
		} else {
			err = u2ferror(status)
		}
	}
	return registerResponse, err
}
