package version

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Version struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	Published   bool   `json:"published"`
	Required    bool   `json:"required"`
}

type Versions []Version

func (vs Versions) Resolve(version string) (v Version, err error) {
	switch {
	case version == "latest" || version == "stable":
		v, err = vs.Latest()
	case version == "edge":
		v = vs[len(vs)-1]
	default:
		v, err = vs.Find(version)
	}
	return
}

// Get all versions as Versions type
func All() (Versions, error) {
	res, err := http.Get("http://convox.s3.amazonaws.com/release/versions.json")

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	vs := Versions{}
	json.Unmarshal(b, &vs)

	return vs, nil
}

// Get latest published version as a string
func Latest() (string, error) {
	vs, err := All()

	if err != nil {
		return "", err
	}

	v, err := vs.Latest()

	return v.Version, err
}

// Get next required or latest published version as a string, based on current version string
func Next(curr string) (string, error) {
	vs, err := All()

	if err != nil {
		return "", err
	}

	v, err := vs.Next(curr)

	return v, err
}

// Append a new version to versions.json file
func AppendVersion(v Version) (Version, error) {
	vs, err := All()

	if err != nil {
		return v, err
	}

	vs = append(vs, v)

	err = putVersions(vs)

	return v, err
}

func (vs Versions) Len() int {
	return len(vs)
}

func (vs Versions) Less(i, j int) bool {
	return vs[i].Version < vs[j].Version
}

func (vs Versions) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

func (v Version) Display() string {
	return fmt.Sprintf("%s (published: %v, required: %v)", v.Version, v.Published, v.Required)
}

func UpdateVersion(v Version) (Version, error) {
	vs, err := All()

	if err != nil {
		return v, err
	}

	for i, _ := range vs {
		if vs[i].Version == v.Version {
			vs[i].Published = v.Published
			vs[i].Required = v.Required

			err := putVersions(vs)

			return vs[i], err
		}
	}

	return v, fmt.Errorf("version %q not found", v.Version)
}

func (vs Versions) Find(version string) (Version, error) {
	for _, v := range vs {
		if v.Version == version {
			return v, nil
		}
	}

	return Version{}, fmt.Errorf("version %q not found", version)
}

func (vs Versions) Latest() (Version, error) {
	for i := len(vs) - 1; i >= 0; i-- {
		v := vs[i]

		if v.Published {
			return v, nil
		}
	}

	return Version{}, fmt.Errorf("no published versions")
}

func (vs Versions) Next(curr string) (string, error) {
	found := false
	published := ""

	for _, v := range vs {
		if v.Version == curr {
			found = true
			continue
		}

		if found && v.Published {
			published = v.Version

			if v.Required {
				return v.Version, nil
			}
		}
	}

	if !found {
		return "", fmt.Errorf("current version %q not found", curr)
	}

	if published != "" {
		return published, nil
	}

	return "", fmt.Errorf("current version %q is latest", curr)
}

// Walk a bucket to create initial versions.json file
func importVersions() (Versions, error) {
	S3 := s3.New(session.New(), &aws.Config{
		Region: aws.String(os.Getenv("AWS_DEFAULT_REGION")),
	})

	res, err := S3.ListObjects(&s3.ListObjectsInput{
		Bucket:    aws.String("convox"),
		Delimiter: aws.String("/"),
		Prefix:    aws.String("release/"),
	})

	if err != nil {
		return nil, err
	}

	vs := Versions{}

	for _, p := range res.CommonPrefixes {
		parts := strings.Split(*p.Prefix, "/")
		version := parts[1]

		if version == "latest" {
			continue
		}

		vs = append(vs, Version{
			Version:   version,
			Published: false,
			Required:  false,
		})
	}

	err = putVersions(vs)

	return vs, err
}

func putVersions(vs Versions) error {
	data, err := json.MarshalIndent(vs, "", "  ")

	if err != nil {
		return err
	}

	S3 := s3.New(session.New(), &aws.Config{
		Region: aws.String(os.Getenv("AWS_DEFAULT_REGION")),
	})

	_, err = S3.PutObject(&s3.PutObjectInput{
		ACL:           aws.String("public-read"),
		Body:          bytes.NewReader(data),
		Bucket:        aws.String("convox"),
		ContentLength: aws.Int64(int64(len(data))),
		Key:           aws.String("release/versions.json"),
	})

	return err
}
