package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) SettingDelete(name string) error {
	return p.s3Delete(p.SettingsBucket, name)
}

func (p *Provider) SettingExists(name string) (bool, error) {
	return p.s3Exists(p.SettingsBucket, name)
}

func (p *Provider) SettingGet(name string) (string, error) {
	exists, err := p.s3Exists(p.SettingsBucket, name)
	if err != nil {
		return "", err
	}

	if !exists {
		return "", errorNotFound(fmt.Sprintf("setting not found: %s", name))
	}

	data, err := p.s3Get(p.SettingsBucket, name)
	if err != nil {
		return "", err
	}

	dec, err := p.SystemDecrypt(data)
	if err != nil {
		return "", err
	}

	return string(dec), nil
}

func (p *Provider) SettingList(opts structs.SettingListOptions) ([]string, error) {
	log := Logger.At("ObjectList").Namespace("prefix=%q", opts.Prefix).Start()

	res, err := p.s3().ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:    aws.String(p.SettingsBucket),
		Delimiter: aws.String("/"),
		Prefix:    aws.String(opts.Prefix),
	})
	if err != nil {
		return nil, log.Error(err)
	}

	objects := []string{}

	for _, item := range res.Contents {
		objects = append(objects, *item.Key)
	}

	return objects, log.Success()
}

func (p *Provider) SettingPut(name, value string) error {
	enc, err := p.SystemEncrypt([]byte(value))
	if err != nil {
		return err
	}

	return p.s3Put(p.SettingsBucket, name, enc, false)
}
