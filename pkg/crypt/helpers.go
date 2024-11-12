package crypt

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/secretbox"
)

const (
	KeySize   = 32
	NonceSize = 24
)

type RandEnvelope struct {
	Nonce      *[NonceSize]byte
	CipherText []byte
}

func OneWay(str string) string {
	enc := sha256.Sum256([]byte(str))
	return base64.StdEncoding.EncodeToString(enc[:])
}

func Encrypt(ekey string, data []byte) (string, error) {
	key, err := decodeKey(ekey)
	if err != nil {
		return "", errors.WithStack(err)
	}

	nonce, err := generateNonce()
	if err != nil {
		return "", errors.WithStack(err)
	}

	var cipherText []byte
	cipherText = secretbox.Seal(cipherText, data, nonce, key)

	envelope := RandEnvelope{
		Nonce:      nonce,
		CipherText: cipherText,
	}

	envelopeJson, err := json.Marshal(envelope)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return base64.StdEncoding.EncodeToString(envelopeJson), nil
}

func Decrypt(ekey string, sealed string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(sealed)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var envelope RandEnvelope

	if err := json.Unmarshal(decoded, &envelope); err != nil {
		return nil, errors.WithStack(err)
	}

	key, err := decodeKey(ekey)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	data := []byte{}

	data, ok := secretbox.Open(nil, envelope.CipherText, envelope.Nonce, key)
	if !ok {
		return nil, errors.WithStack(fmt.Errorf("could not decrypt data"))
	}

	return data, nil
}

func decodeKey(ekey string) (*[KeySize]byte, error) {
	var key [KeySize]byte

	data, err := base64.StdEncoding.DecodeString(ekey)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	copy(key[:], data[0:KeySize])

	return &key, nil
}

func generateNonce() (*[NonceSize]byte, error) {
	nonce := new([NonceSize]byte)
	_, err := io.ReadFull(rand.Reader, nonce[:])
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return nonce, nil
}
