package kaws

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
	"github.com/convox/rack/pkg/structs"
)

func (p *Provider) ObjectDelete(app, key string) error {
	exists, err := p.ObjectExists(app, key)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("object not found: %s", key)
	}

	_, err = p.S3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(p.Bucket),
		Key:    aws.String(p.objectKey(app, key)),
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) ObjectExists(app, key string) (bool, error) {
	_, err := p.S3.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(p.Bucket),
		Key:    aws.String(p.objectKey(app, key)),
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
	res, err := p.S3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(p.Bucket),
		Key:    aws.String(p.objectKey(app, key)),
	})
	if ae, ok := err.(awserr.Error); ok && ae.Code() == "NoSuchKey" {
		return nil, fmt.Errorf("object not found: %s", key)
	}
	if err != nil {
		return nil, err
	}

	return res.Body, nil
}

func (p *Provider) ObjectList(app, prefix string) ([]string, error) {
	res, err := p.S3.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:    aws.String(p.Bucket),
		Delimiter: aws.String("/"),
		Prefix:    aws.String(p.objectKey(app, prefix)),
	})
	if err != nil {
		return nil, err
	}

	objects := []string{}

	for _, item := range res.Contents {
		objects = append(objects, *item.Key)
	}

	return objects, nil
}

// ObjectStore stores an Object
func (p *Provider) ObjectStore(app, key string, r io.Reader, opts structs.ObjectStoreOptions) (*structs.Object, error) {
	if key == "" {
		k, err := generateTempKey()
		if err != nil {
			return nil, err
		}
		key = k
	}

	up := s3manager.NewUploaderWithClient(p.S3)

	req := &s3manager.UploadInput{
		Bucket: aws.String(p.Bucket),
		Key:    aws.String(p.objectKey(app, key)),
		Body:   r,
	}

	if opts.Public != nil && *opts.Public {
		req.ACL = aws.String("public-read")
	}

	res, err := up.Upload(req)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("object://%s/%s", app, key)

	if opts.Public != nil && *opts.Public {
		url = res.Location
	}

	o := &structs.Object{Url: url}

	return o, nil
}

func (p *Provider) objectKey(app, key string) string {
	return fmt.Sprintf("%s/%s", app, key)
}

// func (p *Provider) objectPresignedURL(o *structs.Object, duration time.Duration) (string, error) {
//   ou, err := url.Parse(o.Url)
//   if err != nil {
//     return "", err
//   }

//   if ou.Scheme != "object" {
//     return "", fmt.Errorf("url is not an object: %s", o.Url)
//   }

//   req, _ := p.S3.GetObjectRequest(&s3.GetObjectInput{
//     Bucket: aws.String(p.Bucket),
//     Key:    aws.String(p.objectKey(ou.Hostname(), ou.Path)),
//   })

//   su, err := req.Presign(duration)
//   if err != nil {
//     return "", err
//   }

//   return su, nil
// }

func generateTempKey() (string, error) {
	data := make([]byte, 1024)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)

	return fmt.Sprintf("tmp/%s", hex.EncodeToString(hash[:])[0:30]), nil
}
