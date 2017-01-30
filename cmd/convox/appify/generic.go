package appify

import "fmt"

// GenericApp is a simple type for generating app files
// It's meant to be simple a starting point for more specific types
type GenericApp struct {
	Kind string
}

// Appify generates the files needed for an app
// Must be called after Setup()
func (g *GenericApp) Appify() error {
	if err := writeAsset("Dockerfile", fmt.Sprintf("appify/templates/%s/Dockerfile", g.Kind), nil); err != nil {
		return err
	}

	if err := writeAsset("docker-compose.yml", fmt.Sprintf("appify/templates/%s/docker-compose.yml", g.Kind), nil); err != nil {
		return err
	}

	if err := writeAsset(".dockerignore", fmt.Sprintf("appify/templates/%s/.dockerignore", g.Kind), nil); err != nil {
		return err
	}

	return nil
}

// Setup doesn't do anything. Returns nil to satisfy the Framework interface
func (g *GenericApp) Setup(location string) error {
	return nil
}
