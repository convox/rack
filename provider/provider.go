package provider

import (
	"os"

	"github.com/convox/rack/provider/aws"
	"github.com/convox/rack/provider/local"
	"github.com/convox/rack/structs"
)

// FromEnv returns a new Provider from env vars
func FromEnv() structs.Provider {
	switch os.Getenv("PROVIDER") {
	case "aws":
		return aws.FromEnv()
	case "local":
		return local.FromEnv()
	default:
		return &structs.MockProvider{}
	}
}
