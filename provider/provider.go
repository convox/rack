package provider

import (
	"fmt"
	"os"

	"github.com/convox/rack/provider/aws"
	"github.com/convox/rack/provider/local"
	"github.com/convox/rack/structs"
)

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
		return &structs.MockProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}
