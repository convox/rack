package aws

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/convox/rack/structs"
)

func (p *Provider) ObjectDelete(app, key string) error {
	exists, err := p.ObjectExists(app, key)
	if err != nil {
		return err
	}

	if !exists {
		return errorNotFound(fmt.Sprintf("object not found: %s", key))
	}

	bucket, err := p.appBucket(app)
	if err != nil {
		return err
	}

	_, err = p.s3().DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) ObjectExists(app, key string) (bool, error) {
	bucket, err := p.appBucket(app)
	if err != nil {
		return false, err
	}

	_, err = p.s3().HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err, ok := err.(awserr.Error); ok && err.Code() == "NotFound" {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

// ObjectFetch fetches an Object
func (p *Provider) ObjectFetch(app, key string) (io.ReadCloser, error) {
	bucket, err := p.appBucket(app)
	if err != nil {
		return nil, err
	}

	res, err := p.s3().GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "NoSuchKey" {
		return nil, errorNotFound(fmt.Sprintf("key not found: %s", key))
	}
	if err != nil {
		return nil, err
	}

	return res.Body, nil
}

func (p *Provider) ObjectList(app, prefix string) ([]string, error) {
	log := Logger.At("ObjectList").Namespace("prefix=%q", prefix).Start()

	bucket, err := p.appBucket(app)
	if err != nil {
		return nil, err
	}

	res, err := p.s3().ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Delimiter: aws.String("/"),
		Prefix:    aws.String(prefix),
	})
	if err != nil {
		return nil, log.Error(err)
	}

	objects := []string{}

	for _, item := range res.Contents {
		objects = append(objects, *item.Key)
	}

	log.Success()
	return objects, nil
}

// ObjectStore stores an Object
func (p *Provider) ObjectStore(app, key string, r io.Reader, opts structs.ObjectStoreOptions) (*structs.Object, error) {
	log := Logger.At("ObjectStore").Namespace("app=%q key=%q", app, key).Start()

	if key == "" {
		k, err := generateTempKey()
		if err != nil {
			return nil, log.Error(err)
		}
		key = k
	}

	log = log.Replace("key", key)

	bucket, err := p.appBucket(app)
	if err != nil {
		return nil, log.Error(err)
	}

	up := s3manager.NewUploaderWithClient(p.s3())

	req := &s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   r,
	}

	if opts.Public != nil && *opts.Public {
		req.ACL = aws.String("public-read")
	}

	res, err := up.Upload(req)
	if err != nil {
		return nil, log.Error(err)
	}

	url := fmt.Sprintf("object://%s/%s", app, key)

	if opts.Public != nil && *opts.Public {
		url = res.Location
	}

	o := &structs.Object{Url: url}

	return o, log.Success()
}

func (p *Provider) appBucket(app string) (string, error) {
	if app == "" {
		return p.rackResource("Settings")
	}

	return p.appResource(app, "Settings")
}

func generateTempKey() (string, error) {
	data := make([]byte, 1024)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)

	return fmt.Sprintf("tmp/%s", hex.EncodeToString(hash[:])[0:30]), nil
}
