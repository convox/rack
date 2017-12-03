// +build lambdabinary

package sparta

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	spartaAWS "github.com/mweagle/Sparta/aws"
)

var instanceID string
var currentSlot int
var stackName string
var profileBucket string

const snapshotCount = 3

func nextUploadSlot() int {
	uploadSlot := currentSlot
	currentSlot = (currentSlot + 1) % snapshotCount
	return uploadSlot
}

func init() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	instanceID = fmt.Sprintf("λ-%d", r.Int63())
	currentSlot = 0
	// These correspond to the environment variables that were published
	// into the Lambda environment by the profile decorator
	stackName = os.Getenv(envVarStackName)
	profileBucket = os.Getenv(envVarProfileBucketName)
}

func profileOutputFile(basename string) (*os.File, error) {
	fileName := fmt.Sprintf("%s.%s.profile", basename, instanceID)
	// http://docs.aws.amazon.com/lambda/latest/dg/current-supported-versions.html
	if os.Getenv("AWS_EXECUTION_ENV") != "" {
		fileName = filepath.Join("/tmp", fileName)
	}
	return os.Create(fileName)
}

////////////////////////////////////////////////////////////////////////////////
// Type returned from worker pool uploading profiles to S3
type uploadResult struct {
	err      error
	uploaded bool
}

func (ur *uploadResult) Error() error {
	return ur.err
}
func (ur *uploadResult) Result() interface{} {
	return ur.uploaded
}

func uploadFileTask(uploader *s3manager.Uploader,
	profileType string,
	uploadSlot int,
	localFilePath string) taskFunc {
	return func() workResult {
		fileReader, fileReaderErr := os.Open(localFilePath)
		if fileReaderErr != nil {
			return &uploadResult{err: fileReaderErr}
		}
		defer fileReader.Close()
		defer os.Remove(localFilePath)

		uploadFileName := fmt.Sprintf("%d-%s", uploadSlot, path.Base(localFilePath))
		keyPath := path.Join(profileSnapshotRootKeypathForType(profileType, stackName), uploadFileName)
		uploadInput := &s3manager.UploadInput{
			Bucket: aws.String(profileBucket),
			Key:    aws.String(keyPath),
			Body:   fileReader,
		}
		fmt.Printf("Uploading %s to %s\n", localFilePath, keyPath)
		uploadOutput, uploadErr := uploader.Upload(uploadInput)
		return &uploadResult{
			err:      uploadErr,
			uploaded: uploadOutput != nil,
		}
	}
}

func snapshotProfiles(s3BucketArchive interface{},
	snapshotInterval time.Duration,
	cpuProfileDuration time.Duration,
	profileTypes ...string) {

	publishProfiles := func(cpuProfilePath string) {
		uploadSlot := nextUploadSlot()

		// The session the S3 Uploader will use
		profileLogger, _ := NewLogger("info")
		sess := spartaAWS.NewSession(profileLogger)
		uploader := s3manager.NewUploader(sess)
		uploadTasks := make([]*workTask, 0)

		if cpuProfilePath != "" {
			uploadTasks = append(uploadTasks,
				newWorkTask(uploadFileTask(uploader, "cpu", uploadSlot, cpuProfilePath)))
		}
		for _, eachProfileType := range profileTypes {
			namedProfile := pprof.Lookup(eachProfileType)
			if namedProfile != nil {
				outputProfile, outputFileErr := profileOutputFile(eachProfileType)
				if outputFileErr != nil {
					fmt.Printf("Failed to create output: %s\n", outputFileErr.Error())
				} else {
					namedProfile.WriteTo(outputProfile, 0)
					outputProfile.Close()
					uploadTasks = append(uploadTasks,
						newWorkTask(uploadFileTask(uploader, eachProfileType, uploadSlot, outputProfile.Name())))
				}
			}
		}
		workerPool := newWorkerPool(uploadTasks, 32)
		workerPool.Run()
		ScheduleProfileLoop(s3BucketArchive, snapshotInterval, cpuProfileDuration, profileTypes...)
	}

	if cpuProfileDuration != 0 {
		outputFile, outputFileErr := profileOutputFile("cpu")
		if outputFileErr != nil {
			return
		}
		startErr := pprof.StartCPUProfile(outputFile)
		if startErr != nil {
			fmt.Printf("Failed to start CPU profile: %s\n", startErr.Error())
		}
		time.AfterFunc(cpuProfileDuration, func() {
			pprof.StopCPUProfile()
			closeErr := outputFile.Close()
			if closeErr != nil {
				fmt.Printf("Failed to close CPU profile output: %s\n", closeErr.Error())
			} else {
				publishProfiles(outputFile.Name())
			}
		})

	} else {
		publishProfiles("")
	}
}

// ScheduleProfileLoop installs a profiling loop that pushes profile information
// to S3 for local consumption using a `profile` command that wraps
// pprof
func ScheduleProfileLoop(s3BucketArchive interface{},
	snapshotInterval time.Duration,
	cpuProfileDuration time.Duration,
	profileTypes ...string) {
	time.AfterFunc(snapshotInterval, func() {
		snapshotProfiles(s3BucketArchive, snapshotInterval, cpuProfileDuration, profileTypes...)
	})
}
