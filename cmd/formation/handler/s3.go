package handler

import (
	"fmt"
	"os"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/service/s3"
)

func HandleS3BucketCleanup(req Request) (string, map[string]string, error) {
	defer recoverFailure(req)

	switch req.RequestType {
	case "Create":
		fmt.Println("CREATING BUCKETCLEANUP")
		fmt.Printf("req %+v\n", req)
		return S3BucketCleanupCreate(req)
	case "Update":
		fmt.Println("UPDATING BUCKETCLEANUP")
		fmt.Printf("req %+v\n", req)
		return S3BucketCleanupUpdate(req)
	case "Delete":
		fmt.Println("DELETING BUCKETCLEANUP")
		fmt.Printf("req %+v\n", req)
		return S3BucketCleanupDelete(req)
	}

	return "", nil, fmt.Errorf("unknown RequestType: %s", req.RequestType)
}

func S3BucketCleanupCreate(req Request) (string, map[string]string, error) {
	return req.ResourceProperties["Bucket"].(string) + "-Cleanup", nil, nil
}

func S3BucketCleanupUpdate(req Request) (string, map[string]string, error) {
	return req.ResourceProperties["Bucket"].(string) + "-Cleanup", nil, nil
}

func S3BucketCleanupDelete(req Request) (string, map[string]string, error) {
	bucket := req.ResourceProperties["Bucket"].(string)

	err := cleanupBucket(bucket, S3(req))

	// TODO let the cloudformation finish thinking this deleted
	// but take note so we can figure out why
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return "", nil, nil
	}

	// success
	return "", nil, nil
}

func cleanupBucket(bucket string, S3 *s3.S3) error {
	req := &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	}

	res, err := S3.ListObjectVersions(req)

	if err != nil {
		return err
	}

	for _, d := range res.DeleteMarkers {
		go cleanupBucketObject(bucket, *d.Key, *d.VersionID, S3)
	}

	for _, v := range res.Versions {
		go cleanupBucketObject(bucket, *v.Key, *v.VersionID, S3)
	}

	return nil
}

func cleanupBucketObject(bucket, key, version string, S3 *s3.S3) {
	req := &s3.DeleteObjectInput{
		Bucket:    aws.String(bucket),
		Key:       aws.String(key),
		VersionID: aws.String(version),
	}

	_, err := S3.DeleteObject(req)

	if err != nil {
		fmt.Printf("error: %s\n", err)
	}
}
