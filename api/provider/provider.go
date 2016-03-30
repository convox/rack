package provider

import (
	"fmt"
	"os"

	"github.com/convox/rack/api/provider/aws"
	"github.com/convox/rack/api/structs"
)

var CurrentProvider Provider

type Provider interface {
	AppGet(name string) (*structs.App, error)

	BuildGet(app, id string) (*structs.Build, error)
	BuildDelete(app, id string) (*structs.Build, error)

	CapacityGet() (*structs.Capacity, error)

	InstanceList() (structs.Instances, error)

	ReleaseGet(app, id string) (*structs.Release, error)

	SystemGet() (*structs.System, error)
	SystemSave(system structs.System) error
}

func init() {
	var err error

	switch os.Getenv("PROVIDER") {
	case "aws":
		CurrentProvider, err = aws.NewProvider(os.Getenv("AWS_REGION"), os.Getenv("AWS_ACCESS"), os.Getenv("AWS_SECRET"), os.Getenv("AWS_ENDPOINT"))
	case "test":
		CurrentProvider = TestProvider
	default:
		die(fmt.Errorf("PROVIDER must be one of (aws)"))
	}

	if err != nil {
		die(err)
	}
}

/** package-level functions ************************************************************************/

func AppGet(name string) (*structs.App, error) {
	return CurrentProvider.AppGet(name)
}

func BuildGet(app, id string) (*structs.Build, error) {
	return CurrentProvider.BuildGet(app, id)
}

func BuildDelete(app, id string) (*structs.Build, error) {
	return CurrentProvider.BuildDelete(app, id)
}

func CapacityGet() (*structs.Capacity, error) {
	return CurrentProvider.CapacityGet()
}

func InstanceList() (structs.Instances, error) {
	return CurrentProvider.InstanceList()
}

func ReleaseGet(app, id string) (*structs.Release, error) {
	return CurrentProvider.ReleaseGet(app, id)
}

func SystemGet() (*structs.System, error) {
	return CurrentProvider.SystemGet()
}

func SystemSave(system structs.System) error {
	return CurrentProvider.SystemSave(system)
}

/** helpers ****************************************************************************************/

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
