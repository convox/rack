package s3

import (
	"fmt"
	"mime"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// RollbackFunction called in the event of a stack provisioning failure
type RollbackFunction func(logger *logrus.Logger) error

// CreateS3RollbackFunc creates an S3 rollback function that attempts to delete a previously
// uploaded item. Note that s3ArtifactURL may include a `versionId` query arg
// to denote the specific version to delete.
func CreateS3RollbackFunc(awsSession *session.Session, s3ArtifactURL string) RollbackFunction {
	return func(logger *logrus.Logger) error {
		logger.WithFields(logrus.Fields{
			"URL": s3ArtifactURL,
		}).Info("Deleting S3 object")
		artifactURLParts, artifactURLPartsErr := url.Parse(s3ArtifactURL)
		if nil != artifactURLPartsErr {
			return artifactURLPartsErr
		}
		// Bucket is the first component
		s3Bucket := strings.Split(artifactURLParts.Host, ".")[0]
		s3Client := s3.New(awsSession)
		params := &s3.DeleteObjectInput{
			Bucket: aws.String(s3Bucket),
			Key:    aws.String(artifactURLParts.Path),
		}
		versionID := artifactURLParts.Query().Get("versionId")
		if "" != versionID {
			params.VersionId = aws.String(versionID)
		}
		_, err := s3Client.DeleteObject(params)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"Error": err,
			}).Warn("Failed to delete S3 item during rollback cleanup")
		}
		return err
	}
}

// UploadLocalFileToS3 takes a local path and uploads the content at localPath
// to the given S3Bucket and KeyPrefix.  The final S3 keyname is the S3KeyPrefix+
// the basename of the localPath.
func UploadLocalFileToS3(localPath string,
	awsSession *session.Session,
	S3Bucket string,
	S3KeyName string,
	logger *logrus.Logger) (string, error) {

	// Then do the actual work
	reader, err := os.Open(localPath)
	if nil != err {
		return "", fmt.Errorf("Failed to open local archive for S3 upload: %s", err.Error())
	}
	uploadInput := &s3manager.UploadInput{
		Bucket:      &S3Bucket,
		Key:         &S3KeyName,
		ContentType: aws.String(mime.TypeByExtension(path.Ext(localPath))),
		Body:        reader,
	}
	// If we can get the current working directory, let's try and strip
	// it from the path just to keep the log statement a bit shorter
	logPath := localPath
	cwd, cwdErr := os.Getwd()
	if cwdErr == nil {
		logPath = strings.TrimPrefix(logPath, cwd)
		if logPath != localPath {
			logPath = fmt.Sprintf(".%s", logPath)
		}
	}
	logger.WithFields(logrus.Fields{
		"Path":   logPath,
		"Bucket": S3Bucket,
		"Key":    S3KeyName,
	}).Info("Uploading local file to S3")

	uploader := s3manager.NewUploader(awsSession)
	result, err := uploader.Upload(uploadInput)
	if nil != err {
		return "", err
	}
	if result.VersionID != nil {
		logger.WithFields(logrus.Fields{
			"URL":       result.Location,
			"VersionID": string(*result.VersionID),
		}).Debug("S3 upload complete")
	} else {
		logger.WithFields(logrus.Fields{
			"URL": result.Location,
		}).Debug("S3 upload complete")
	}
	locationURL := result.Location
	if nil != result.VersionID {
		// http://docs.aws.amazon.com/AmazonS3/latest/dev/RetrievingObjectVersions.html
		locationURL = fmt.Sprintf("%s?versionId=%s", locationURL, string(*result.VersionID))
	}
	return locationURL, nil
}

// BucketVersioningEnabled determines if a given S3 bucket has object
// versioning enabled.
func BucketVersioningEnabled(awsSession *session.Session,
	S3Bucket string,
	logger *logrus.Logger) (bool, error) {

	s3Svc := s3.New(awsSession)
	params := &s3.GetBucketVersioningInput{
		Bucket: aws.String(S3Bucket), // Required
	}
	versioningEnabled := false
	resp, err := s3Svc.GetBucketVersioning(params)
	if err == nil && resp != nil && resp.Status != nil {
		// What's the versioning policy?
		logger.WithFields(logrus.Fields{
			"VersionPolicy": *resp,
			"BucketName":    S3Bucket,
		}).Debug("Bucket version policy")
		versioningEnabled = (strings.ToLower(*resp.Status) == "enabled")
	}
	return versioningEnabled, err
}
