package aws

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/convox/rack/structs"
)

func (p *AWSProvider) ObjectDelete(app, key string) error {
	exists, err := p.ObjectExists(app, key)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("no such object: %s", key)
	}

	bucket, err := p.appResource(app, "Settings")
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

func (p *AWSProvider) ObjectExists(app, key string) (bool, error) {
	bucket, err := p.appResource(app, "Settings")
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
func (p *AWSProvider) ObjectFetch(app, key string) (io.ReadCloser, error) {
	bucket, err := p.appResource(app, "Settings")
	if err != nil {
		return nil, err
	}

	res, err := p.s3().GetObject(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "NoSuchKey" {
		return nil, errorNotFound(fmt.Sprintf("no such key: %s", key))
	}
	if err != nil {
		return nil, err
	}

	return res.Body, nil
}

func (p *AWSProvider) ObjectList(app, prefix string) ([]string, error) {
	log := Logger.At("ObjectList").Namespace("prefix=%q", prefix).Start()

	bucket, err := p.appResource(app, "Settings")
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
func (p *AWSProvider) ObjectStore(app, key string, r io.Reader, opts structs.ObjectStoreOptions) (*structs.Object, error) {
	log := Logger.At("ObjectStore").Namespace("app=%q key=%q public=%t", app, key, opts.Public).Start()

	if key == "" {
		k, err := generateTempKey()
		if err != nil {
			return nil, log.Error(err)
		}
		key = k
	}

	log = log.Replace("key", key)

	bucket, err := p.appResource(app, "Settings")
	if err != nil {
		return nil, err
	}

	mreq := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	if opts.Public != nil && *opts.Public {
		mreq.ACL = aws.String("public-read")
	}

	mres, err := p.s3().CreateMultipartUpload(mreq)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// buf := make([]byte, 5*1024*1024)
	buf := make([]byte, 10*1024*1024)
	i := 1
	parts := []*s3.CompletedPart{}

	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return nil, err
		}

		res, err := p.s3().UploadPart(&s3.UploadPartInput{
			Body:          bytes.NewReader(buf[0:n]),
			Bucket:        aws.String(bucket),
			ContentLength: aws.Int64(int64(n)),
			Key:           aws.String(key),
			PartNumber:    aws.Int64(int64(i)),
			UploadId:      mres.UploadId,
		})
		if err != nil {
			log.Error(err)
			return nil, err
		}

		parts = append(parts, &s3.CompletedPart{
			ETag:       res.ETag,
			PartNumber: aws.Int64(int64(i)),
		})

		i++
	}

	res, err := p.s3().CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: parts,
		},
		UploadId: mres.UploadId,
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}

	log.Success()

	url := fmt.Sprintf("object://%s/%s", app, key)

	if opts.Public != nil && *opts.Public {
		url = *res.Location
	}

	o := &structs.Object{Url: url}

	return o, nil
}

func generateTempKey() (string, error) {
	data := make([]byte, 1024)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)

	return fmt.Sprintf("tmp/%s", hex.EncodeToString(hash[:])[0:30]), nil
}
