package aws

import "os"

func releasesTable(app string) string {
	return os.Getenv("DYNAMO_RELEASES")
}
