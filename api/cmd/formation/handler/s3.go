package handler

import "fmt"

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
	return req.PhysicalResourceId, nil, nil
}

func S3BucketCleanupDelete(req Request) (string, map[string]string, error) {
	// bucket := req.ResourceProperties["Bucket"].(string)

	// err := cleanupBucket(bucket, S3(req))

	// // TODO let the cloudformation finish thinking this deleted
	// // but take note so we can figure out why
	// if err != nil {
	//   fmt.Printf("error: %s\n", err)
	//   return req.PhysicalResourceId, nil, nil
	// }

	// success
	return req.PhysicalResourceId, nil, nil
}
