package provider

import (
	"os"

	"github.com/convox/rack/provider/aws"
	"github.com/convox/rack/provider/local"
	"github.com/convox/rack/structs"
)

// FromEnv returns a new Provider from env vars
func FromEnv() structs.Provider {
	return FromName(os.Getenv("PROVIDER"))
}

func FromName(name string) structs.Provider {
	switch name {
	case "aws":
		return aws.FromEnv()
	case "local":
		return local.FromEnv()
	default:
		return &structs.MockProvider{}
	}
}
