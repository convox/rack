package provider

import (
	"fmt"
	"os"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/provider/aws"
	"github.com/convox/rack/provider/k8s"
	"github.com/convox/rack/provider/klocal"
	"github.com/convox/rack/structs"
)

var Mock = &structs.MockProvider{}

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
