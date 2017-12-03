package aws

import (
	"fmt"

	"github.com/convox/rack/api/crypt"
)

func (p *AWSProvider) SettingGet(name string) (string, error) {
	exists, err := p.s3Exists(p.SettingsBucket, name)
	if err != nil {
		return "", err
	}

	if !exists {
		return "", fmt.Errorf("no such setting: %s", name)
	}

	data, err := p.s3Get(p.SettingsBucket, name)
	if err != nil {
		return "", err
	}

	dec, err := crypt.New().Decrypt(p.EncryptionKey, data)
	if err != nil {
		return "", err
	}

	return string(dec), nil
}

func (p *AWSProvider) SettingPut(name, value string) error {
	enc, err := crypt.New().Encrypt(p.EncryptionKey, []byte(value))
	if err != nil {
		return err
	}

	return p.s3Put(p.SettingsBucket, name, enc, false)
}
