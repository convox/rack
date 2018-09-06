package provider

import (
	"fmt"
	"os"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/provider/aws"
	"github.com/convox/rack/provider/base"
	"github.com/convox/rack/provider/k8s"
	"github.com/convox/rack/provider/kaws"
	"github.com/convox/rack/provider/klocal"
)

var Mock = &structs.MockProvider{}

// make sure base provider stays in sync
var baseProvider structs.Provider = &base.Provider{}

// FromEnv returns a new Provider from env vars
func FromEnv() (structs.Provider, error) {
	return FromName(os.Getenv("PROVIDER"))
}

func FromName(name string) (structs.Provider, error) {
	switch name {
	case "aws":
		return aws.FromEnv()
	case "k8s":
		return k8s.FromEnv()
	case "kaws":
		return kaws.FromEnv()
	case "klocal":
		return klocal.FromEnv()
	case "test":
		return Mock, nil
	case "":
		return nil, fmt.Errorf("PROVIDER not set")
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}
