package crypt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/kms"
	"golang.org/x/crypto/nacl/secretbox"
)

const (
	KeyLength   = 32
	NonceLength = 24
)

type Crypt struct {
	AwsRegion string
	AwsAccess string
	AwsSecret string
	AwsToken  string
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

func New(region, access, secret string) *Crypt {
	return &Crypt{
		AwsRegion: region,
		AwsAccess: access,
		AwsSecret: secret,
	}
}

func NewIam(role string) (*Crypt, error) {
	res, err := http.Get("http://169.254.169.254/latest/dynamic/instance-identity/document")

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	id, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	var instance Instance

	err = json.Unmarshal(id, &instance)

	if err != nil {
		return nil, err
	}

	res, err = http.Get(fmt.Sprintf("http://169.254.169.254/latest/meta-data/iam/security-credentials/%s", role))

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	rd, err := ioutil.ReadAll(res.Body)

	var r Role

	err = json.Unmarshal(rd, &r)

	if err != nil {
		return nil, err
	}

	crypt := &Crypt{
		AwsRegion: instance.Region,
		AwsAccess: r.Access,
		AwsSecret: r.Secret,
		AwsToken:  r.Token,
	}

	return crypt, nil
}

func (c *Crypt) Encrypt(keyArn string, dec []byte) ([]byte, error) {
	req := &kms.GenerateDataKeyInput{
		KeyID:         aws.String(keyArn),
		NumberOfBytes: aws.Long(KeyLength),
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
	res, err := KMS(c).GenerateRandom(&kms.GenerateRandomInput{NumberOfBytes: aws.Long(NonceLength)})

	if err != nil {
		return nil, err
	}

	return res.Plaintext, nil
}
