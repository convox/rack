package main

import (
	"encoding/json"
	"fmt"

	"github.com/convox/env/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/env/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/kms"
	"github.com/convox/env/Godeps/_workspace/src/golang.org/x/crypto/nacl/secretbox"
)

const (
	KeyLength   = 32
	NonceLength = 24
)

type Crypt struct {
	AwsRegion string
	AwsAccess string
	AwsSecret string
}

type EncryptedData struct {
	Ciphertext   []byte `json:"c"`
	EncryptedKey []byte `json:"k"`
	Nonce        []byte `json:"n"`
}

func (c *Crypt) Encrypt(keyArn string, dec []byte) (*EncryptedData, error) {
	req := &kms.GenerateDataKeyRequest{
		KeyID:         aws.String(keyArn),
		NumberOfBytes: aws.Integer(KeyLength),
	}

	res, err := c.kms().GenerateDataKey(req)

	if err != nil {
		return nil, err
	}

	var key [KeyLength]byte
	copy(key[:], res.Plaintext[0:KeyLength])

	rand, err := c.generateNonce()

	if err != nil {
		return nil, err
	}

	var nonce [NonceLength]byte
	copy(nonce[:], rand[0:NonceLength])

	var enc []byte
	enc = secretbox.Seal(enc, dec, &nonce, &key)

	data := &EncryptedData{
		Ciphertext:   enc,
		EncryptedKey: res.CiphertextBlob,
		Nonce:        nonce[:],
	}

	return data, nil
}

func (c *Crypt) Decrypt(keyArn string, data *EncryptedData) ([]byte, error) {
	req := &kms.DecryptRequest{
		CiphertextBlob: data.EncryptedKey,
	}

	res, err := c.kms().Decrypt(req)

	if err != nil {
		return nil, err
	}

	var key [KeyLength]byte
	copy(key[:], res.Plaintext[0:KeyLength])

	var nonce [NonceLength]byte
	copy(nonce[:], data.Nonce[0:NonceLength])

	var dec []byte
	dec, ok := secretbox.Open(dec, data.Ciphertext, &nonce, &key)

	if !ok {
		return nil, fmt.Errorf("failed decryption")
	}

	return dec, nil
}

func (c *Crypt) kms() *kms.KMS {
	return kms.New(aws.Creds(c.AwsAccess, c.AwsSecret, ""), c.AwsRegion, nil)
}

func (c *Crypt) generateNonce() ([]byte, error) {
	res, err := c.kms().GenerateRandom(&kms.GenerateRandomRequest{NumberOfBytes: aws.Integer(NonceLength)})

	if err != nil {
		return nil, err
	}

	return res.Plaintext, nil
}

func (ed *EncryptedData) Marshal() ([]byte, error) {
	return json.Marshal(ed)
}

func UnmarshalEncryptedData(data []byte) (*EncryptedData, error) {
	var ed *EncryptedData

	err := json.Unmarshal(data, &ed)

	if err != nil {
		return nil, err
	}

	return ed, nil
}
