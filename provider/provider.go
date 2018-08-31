package provider

import (
	"fmt"
	"os"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/provider/aws"
	"github.com/convox/rack/provider/local"
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
	case "local":
		return local.FromEnv()
	case "test":
		return Mock, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}
