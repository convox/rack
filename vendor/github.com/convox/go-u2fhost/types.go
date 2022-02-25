package u2fhost

// A Device is the interface for performing registration and authentication operations.
type Device interface {
	Open() error
	Close()
	Version() (string, error)
	Register(*RegisterRequest) (*RegisterResponse, error)
	Authenticate(*AuthenticateRequest) (*AuthenticateResponse, error)
}

// A RegisterRequest struct is used when attempting to register a new U2F device.
type RegisterRequest struct {
	// A random string which the new device will sign, this should be
	// provided by the server.
	Challenge string

	// The AppId can be provided by the server, but if not it should
	// be provided by the client.
	// For more information on AppId and Facets see https://fidoalliance.org/specs/fido-u2f-v1.0-ps-20141009/fido-appid-and-facets-ps-20141009.html#the-appid-and-facetid-assertions
	AppId string

	// The Facet should be provided by the client.
	// For more information on AppId and Facets see https://fidoalliance.org/specs/fido-u2f-v1.0-ps-20141009/fido-appid-and-facets-ps-20141009.html#the-appid-and-facetid-assertions
	Facet string

	// Optional channel id public key, mutually exclusive with setting ChannelIdUnused to true.
	ChannelIdPublicKey *JSONWebKey

	// Only set to true if the client supports channel id, but the server does not.
	// Setting to true is mutually exclusive with providing a ChannelIdPublicKey.
	ChannelIdUnused bool
}

// A response from a Register operation.
// The response fields are typically passed back to the server.
type RegisterResponse struct {
	// Base64 encoded registration data.
	RegistrationData string `json:"registrationData"`
	// Base64 encoded client data.
	ClientData string `json:"clientData"`
}

// An AuthenticateRequest is used when attempting to sign the challenge with a
// previously registered U2F device.
type AuthenticateRequest struct {
	// A string to sign. If used for authentication it should be a random string,
	// but could also be used to sign other kinds of data (ex: commit sha).
	Challenge string

	// The AppId can be provided by the server, but if not it should
	// be provided by the client.
	// For more information on AppId and Facets see https://fidoalliance.org/specs/fido-u2f-v1.0-ps-20141009/fido-appid-and-facets-ps-20141009.html#the-appid-and-facetid-assertions
	AppId string

	// The Facet should be provided by the client.
	// For more information on AppId and Facets see https://fidoalliance.org/specs/fido-u2f-v1.0-ps-20141009/fido-appid-and-facets-ps-20141009.html#the-appid-and-facetid-assertions
	Facet string

	// The base64 encoded key handle that was returned in the RegistrationData field of the RegisterResponse.
	KeyHandle string

	// Optional channel id public key, mutually exclusive with setting ChannelIdUnused to true.
	ChannelIdPublicKey *JSONWebKey

	// Only set to true if the client supports channel id, but the server does not.
	// Setting to true is mutually exclusive with providing a ChannelIdPublicKey.
	ChannelIdUnused bool

	// Optional boolean (defaults to false) that when true, will not attempt to
	// sign the challenge, and will only return the status.
	// This can be used to determine if a U2F device matches any of the provided key handles
	// before attempting to prompt the user to activate their devices.
	CheckOnly bool

	// Optional boolean (defaults to false) to use WebAuthn authentication with U2f
	// devices
	WebAuthn bool
}

// A response from an Authenticate operation.
// The response fields are typically passed back to the server.
type AuthenticateResponse struct {
	KeyHandle         string `json:"keyHandle"`
	ClientData        string `json:"clientData"`
	SignatureData     string `json:"signatureData"`
	AuthenticatorData string `json:"authenticatorData,omitempty"`
}

type JSONWebKey struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type clientData struct {
	Typ                string      `json:"typ,omitempty"`
	Type               string      `json:"type,omitempty"`
	Challenge          string      `json:"challenge"`
	ChannelIdPublicKey interface{} `json:"cid_pubkey,omitempty"`
	Origin             string      `json:"origin"`
}
