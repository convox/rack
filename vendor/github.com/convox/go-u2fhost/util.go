package u2fhost

import (
	sha256pkg "crypto/sha256"
	"encoding/base64"
	"fmt"
)

func websafeEncode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func websafeDecode(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}

func sha256(data []byte) []byte {
	sha256Instance := sha256pkg.New()
	sha256Instance.Write(data)
	return sha256Instance.Sum(nil)
}

// Returns the value for the ChannelId public key, can be nil, a JSONWebKey, or the string "unused".
// For more info see cid_pubkey https://fidoalliance.org/specs/fido-u2f-v1.0-nfc-bt-amendment-20150514/fido-u2f-raw-message-formats.html#idl-def-ClientData
func channelIdPublicKey(jwk *JSONWebKey, unused bool) (interface{}, error) {
	if unused && jwk != nil {
		return nil, fmt.Errorf("ChannelIdPublicKey was supplied, but ChannelIdUnsed was set to true.")
	}
	if jwk != nil {
		return jwk, nil
	}
	if unused {
		//
		return "unused", nil
	}
	return nil, nil
}

func u2ferror(err uint16) error {
	if err == u2fStatusConditionsNotSatisfied {
		return &TestOfUserPresenceRequiredError{}
	} else if err == u2fStatusWrongData {
		return &BadKeyHandleError{}
	}
	return fmt.Errorf("U2FError: 0x%02x", err)
}
