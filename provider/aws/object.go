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

func (p *AWSProvider) ObjectDelete(key string) error {
	if !p.ObjectExists(key) {
		return fmt.Errorf("no such object: %s", key)
	}

	_, err := p.s3().DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(p.SettingsBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *AWSProvider) ObjectExists(key string) bool {
	_, err := p.s3().HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(p.SettingsBucket),
		Key:    aws.String(key),
	})
	if err, ok := err.(awserr.Error); ok && err.Code() == "NotFound" {
		return false
	}
	return true
}

// ObjectFetch fetches an Object
func (p *AWSProvider) ObjectFetch(key string) (io.ReadCloser, error) {
	res, err := p.s3().GetObject(&s3.GetObjectInput{
		Bucket: aws.String(p.SettingsBucket),
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

func (p *AWSProvider) ObjectList(prefix string) ([]string, error) {
	log := Logger.At("ObjectList").Namespace("prefix=%q", prefix).Start()

	res, err := p.s3().ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:    aws.String(p.SettingsBucket),
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
func (p *AWSProvider) ObjectStore(key string, r io.Reader, opts structs.ObjectOptions) (string, error) {
	log := Logger.At("ObjectStore").Namespace("key=%q public=%t", key, opts.Public).Start()

	if key == "" {
		k, err := generateTempKey()
		if err != nil {
			log.Error(err)
			return "", err
		}
		key = k
	}

	log = log.Replace("key", key)

	mreq := &s3.CreateMultipartUploadInput{
		Bucket: aws.String(p.SettingsBucket),
		Key:    aws.String(key),
	}

	if opts.Public {
		mreq.ACL = aws.String("public-read")
	}

	mres, err := p.s3().CreateMultipartUpload(mreq)
	if err != nil {
		log.Error(err)
		return "", err
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
			return "", err
		}

		res, err := p.s3().UploadPart(&s3.UploadPartInput{
			Body:          bytes.NewReader(buf[0:n]),
			Bucket:        aws.String(p.SettingsBucket),
			ContentLength: aws.Int64(int64(n)),
			Key:           aws.String(key),
			PartNumber:    aws.Int64(int64(i)),
			UploadId:      mres.UploadId,
		})
		if err != nil {
			log.Error(err)
			return "", err
		}

		parts = append(parts, &s3.CompletedPart{
			ETag:       res.ETag,
			PartNumber: aws.Int64(int64(i)),
		})

		i++
	}

	res, err := p.s3().CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket: aws.String(p.SettingsBucket),
		Key:    aws.String(key),
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: parts,
		},
		UploadId: mres.UploadId,
	})
	if err != nil {
		log.Error(err)
		return "", err
	}

	log.Success()

	url := fmt.Sprintf("object:///%s", key)

	if opts.Public {
		url = *res.Location
	}

	return url, nil
}

func generateTempKey() (string, error) {
	data := make([]byte, 1024)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)

	return fmt.Sprintf("tmp/%s", hex.EncodeToString(hash[:])[0:30]), nil
}
