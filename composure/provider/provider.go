package provider

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/convox/rack/composure/structs"
)

var CurrentProvider Provider

type Provider interface {
	ImageBuild(string, string, string) error
	ImageInspect(string) (map[string]string, error)
	ImagePull(string) error
	ImagePush(string, string) error
	ImageTag(string, string) error

	NetworkInspect() (string, error)

	ManifestBuild(string, string) (map[string]string, error)
	ManifestLoad(string, string) (*structs.Manifest, error)
	ManifestPush(string, string, string, string) error
	ManifestRun(string, string) error

	ProcessRun(string, []string, string, []string, map[string]string) error

	ProjectName(string) (string, error)
}

func init() {
	var err error

	switch os.Getenv("PROVIDER") {
	// case "docker":
	// 	CurrentProvider, err = docker.NewProvider()
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

func ImageBuild(path, dockerfile, tag string) error {
	return CurrentProvider.ImageBuild(path, dockerfile, tag)
}

func ImageInspect(tag string) (map[string]string, error) {
	return CurrentProvider.ImageInspect(tag)
}

func ImagePull(name string) error {
	return CurrentProvider.ImagePull(name)
}

func ImagePush(name string, url string) error {
	return CurrentProvider.ImagePush(name, url)
}

func ImageTag(name, tag string) error {
	return CurrentProvider.ImageTag(name, tag)
}

func NetworkInspect() (string, error) {
	return CurrentProvider.NetworkInspect()
}

func ManifestBuild(path, manifestfile string) (map[string]string, error) {
	return CurrentProvider.ManifestBuild(path, manifestfile)
}

func ManifestLoad(path, manifestfile string) (*structs.Manifest, error) {
	return CurrentProvider.ManifestLoad(path, manifestfile)
}

func ManifestPush(path, manifestfile, registry, repository string) error {
	return CurrentProvider.ManifestPush(path, manifestfile, registry, repository)
}

func ManifestRun(path, manifestfile string) error {
	return CurrentProvider.ManifestRun(path, manifestfile)
}

func ProcessRun(tag string, args []string, name string, ports []string, env map[string]string) error {
	return CurrentProvider.ProcessRun(tag, args, name, ports, env)
}

func ProjectName(path string) (string, error) {
	return CurrentProvider.ProjectName(path)
}

/** helpers ****************************************************************************************/

func die(err error) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}

var randomAlphabet = []rune("abcdefghijklmnopqrstuvwxyz")

func randomString(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = randomAlphabet[rand.Intn(len(randomAlphabet))]
	}
	return prefix + string(b)
}
