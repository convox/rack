package crypt

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"golang.org/x/crypto/nacl/secretbox"
)

const (
	KeyLength   = 32
	NonceLength = 24
)

type Crypt struct {
	Region string
}

type Instance struct {
	Region string `json::"region"`
}

type Role struct {
	Access string `json:"AccessKeyId"`
	Secret string `json:"SecretAccessKey"`
	Token  string `json:"Token"`
}

type Envelope struct {
	Ciphertext   []byte `json:"c"`
	EncryptedKey []byte `json:"k"`
	Nonce        []byte `json:"n"`
}

func KMS(c *Crypt) *kms.KMS {
	return kms.New(session.New(), &aws.Config{
		Region: aws.String(c.Region),
	})
}

func New() *Crypt {
	return &Crypt{Region: os.Getenv("AWS_REGION")}
}

func (c *Crypt) Encrypt(keyArn string, dec []byte) ([]byte, error) {
	req := &kms.GenerateDataKeyInput{
		KeyId:         aws.String(keyArn),
		NumberOfBytes: aws.Int64(KeyLength),
	}

	res, err := KMS(c).GenerateDataKey(req)

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

	e := &Envelope{
		Ciphertext:   enc,
		EncryptedKey: res.CiphertextBlob,
		Nonce:        nonce[:],
	}

	return json.Marshal(e)
}

func (c *Crypt) Decrypt(keyArn string, data []byte) ([]byte, error) {
	var e *Envelope
	err := json.Unmarshal(data, &e)

	if err != nil {
		return nil, err
	}

	req := &kms.DecryptInput{
		CiphertextBlob: e.EncryptedKey,
	}

	res, err := KMS(c).Decrypt(req)

	if err != nil {
		return nil, err
	}

	var key [KeyLength]byte
	copy(key[:], res.Plaintext[0:KeyLength])

	var nonce [NonceLength]byte
	copy(nonce[:], e.Nonce[0:NonceLength])

	var dec []byte
	dec, ok := secretbox.Open(dec, e.Ciphertext, &nonce, &key)

	if !ok {
		return nil, fmt.Errorf("failed decryption")
	}

	return dec, nil
}

func (c *Crypt) generateNonce() ([]byte, error) {
	res, err := KMS(c).GenerateRandom(&kms.GenerateRandomInput{NumberOfBytes: aws.Int64(NonceLength)})

	if err != nil {
		return nil, err
	}

	return res.Plaintext, nil
}
