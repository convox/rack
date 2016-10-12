package aws

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"golang.org/x/crypto/nacl/secretbox"
)

const (
	KeyLength   = 32
	NonceLength = 24
)

type envelope struct {
	Ciphertext   []byte `json:"c"`
	EncryptedKey []byte `json:"k"`
	Nonce        []byte `json:"n"`
}

func (p *AWSProvider) KeyDecrypt(data []byte) ([]byte, error) {
	var e *envelope

	err := json.Unmarshal(data, &e)
	if err != nil {
		return nil, err
	}

	if len(e.EncryptedKey) == 0 {
		return nil, fmt.Errorf("invalid ciphertext")
	}

	res, err := p.kms().Decrypt(&kms.DecryptInput{
		CiphertextBlob: e.EncryptedKey,
	})
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

func (p *AWSProvider) KeyEncrypt(data []byte) ([]byte, error) {
	req := &kms.GenerateDataKeyInput{
		KeyId:         aws.String(p.EncryptionKey),
		NumberOfBytes: aws.Int64(KeyLength),
	}

	res, err := p.kms().GenerateDataKey(req)
	if err != nil {
		return nil, err
	}

	var key [KeyLength]byte
	copy(key[:], res.Plaintext[0:KeyLength])

	nres, err := p.kms().GenerateRandom(&kms.GenerateRandomInput{
		NumberOfBytes: aws.Int64(NonceLength),
	})
	if err != nil {
		return nil, err
	}

	var nonce [NonceLength]byte
	copy(nonce[:], nres.Plaintext[0:NonceLength])

	var enc []byte

	enc = secretbox.Seal(enc, data, &nonce, &key)

	e := &envelope{
		Ciphertext:   enc,
		EncryptedKey: res.CiphertextBlob,
		Nonce:        nonce[:],
	}

	return json.Marshal(e)
}
