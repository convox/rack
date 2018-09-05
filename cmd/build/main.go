package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/convox/rack/pkg/build"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
)

var (
	flagApp         string
	flagAuth        string
	flagCache       string
	flagDevelopment string
	flagGeneration  string
	flagID          string
	flagManifest    string
	flagMethod      string
	flagPush        string
	flagRack        string
	flagUrl         string

	currentBuild    *structs.Build
	currentLogs     string
	currentManifest string

	rack *sdk.Client
)

func main() {
	if err := execute(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}

func execute() error {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	fs.StringVar(&flagApp, "app", "example", "app name")
	fs.StringVar(&flagAuth, "auth", "", "docker auth data (json)")
	fs.StringVar(&flagCache, "cache", "true", "use docker cache")
	fs.StringVar(&flagDevelopment, "development", "false", "create a development build")
	fs.StringVar(&flagGeneration, "generation", "", "app generation")
	fs.StringVar(&flagID, "id", "latest", "build id")
	fs.StringVar(&flagManifest, "manifest", "", "path to app manifest")
	fs.StringVar(&flagMethod, "method", "", "source method")
	fs.StringVar(&flagPush, "push", "", "push to registry")
	fs.StringVar(&flagRack, "rack", "convox", "rack name")
	fs.StringVar(&flagUrl, "url", "", "source url")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	if v := os.Getenv("BUILD_APP"); v != "" {
		flagApp = v
	}

	if v := os.Getenv("BUILD_AUTH"); v != "" {
		flagAuth = v
	}

	if v := os.Getenv("BUILD_DEVELOPMENT"); v != "" {
		flagDevelopment = v
	}

	if v := os.Getenv("BUILD_GENERATION"); v != "" {
		flagGeneration = v
	}

	if v := os.Getenv("BUILD_ID"); v != "" {
		flagID = v
	}

	if v := os.Getenv("BUILD_MANIFEST"); v != "" {
		flagManifest = v
	}

	if v := os.Getenv("BUILD_PUSH"); v != "" {
		flagPush = v
	}

	if v := os.Getenv("BUILD_RACK"); v != "" {
		flagRack = v
	}

	if v := os.Getenv("BUILD_URL"); v != "" {
		flagUrl = v
	}

	opts := build.Options{
		App:         flagApp,
		Auth:        flagAuth,
		Cache:       flagCache == "true",
		Development: flagDevelopment == "true",
		Generation:  flagGeneration,
		Id:          flagID,
		Manifest:    flagManifest,
		Push:        flagPush,
		Rack:        flagRack,
		Source:      flagUrl,
	}

	b, err := build.New(opts)
	if err != nil {
		return err
	}

	if err := b.Execute(); err != nil {
		return err
	}

	return nil
}
