package models

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/s3"
)

func SettingGet(name string) (string, error) {
	req := &s3.GetObjectRequest{
		Bucket: aws.String(os.Getenv("S3_BUCKET")),
		Key:    aws.String(name),
	}

	res, err := S3.GetObject(req)

	if err != nil && err.Error() == "The specified key does not exist." {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	value, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return "", err
	}

	fmt.Printf("value %+v\n", value)

	return string(value), nil
}

func SettingSet(name, value string) error {
	req := &s3.PutObjectRequest{
		Body:          ioutil.NopCloser(strings.NewReader(value)),
		Bucket:        aws.String(os.Getenv("S3_BUCKET")),
		ContentLength: aws.Long(int64(len(value))),
		Key:           aws.String(name),
	}

	_, err := S3.PutObject(req)

	return err
}
