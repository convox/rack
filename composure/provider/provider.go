package provider

import (
	"fmt"
	"os"

	"github.com/convox/rack/composure/provider/docker"
	"github.com/convox/rack/composure/structs"
)

var CurrentProvider Provider

type Provider interface {
	Load(path string) (*structs.Manifest, error)

	Pull(*structs.Manifest) error
}

func init() {
	var err error

	switch os.Getenv("PROVIDER") {
	case "docker":
		CurrentProvider, err = docker.NewProvider()
	case "test":
		CurrentProvider = TestProvider
	default:
		die(fmt.Errorf("PROVIDER must be one of (docker, test)"))
	}

	if err != nil {
		die(err)
	}
}

/** package-level functions ************************************************************************/

func Load(path string) (*structs.Manifest, error) {
	return CurrentProvider.Load(path)
}

func Pull(m *structs.Manifest) error {
	return CurrentProvider.Pull(m)
}

/** helpers ****************************************************************************************/

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
