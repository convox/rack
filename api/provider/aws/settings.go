package aws

import (
	"bytes"
	"os"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/s3"
)

func (p *AWSProvider) SettingsSave(filename string, data []byte, public bool) error {
	req := &s3.PutObjectInput{
		Body:          bytes.NewReader(data),
		Bucket:        aws.String(os.Getenv("SETTINGS_BUCKET")),
		ContentLength: aws.Int64(int64(len(data))),
		Key:           aws.String(filename),
	}

	if public {
		req.ACL = aws.String("public-read")
	}

	_, err := p.s3().PutObject(req)

	return err
}
