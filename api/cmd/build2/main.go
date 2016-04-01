package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/convox/rack/api/manifest"
	"github.com/convox/rack/client"
)

var (
	manifestPath    string
	app             string
	cache           = true
	registryAddress string
	buildId         string
	repository      string
	rackClient      = client.New(os.Getenv("RACK_HOST"), os.Getenv("RACK_PASSWORD"), "build")
)

func init() {
	app = os.Getenv("APP")
	buildId = os.Getenv("BUILD")
	registryAddress = os.Getenv("REGISTRY_ADDRESS")
	repository = os.Getenv("REPOSITORY")

	manifestPath = os.Getenv("MANIFEST_PATH")
	if manifestPath == "" {
		manifestPath = "docker-compose.yml"
	}

	if os.Getenv("NO_CACHE") != "" {
		cache = false
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: build2 <src>")
		os.Exit(1)
	}

	src := os.Args[1]

	if src == "-" {
		extractTar()
	} else {
		cloneGit(src)
	}

	writeDockerAuth()

	m, err := manifest.Read("src", manifestPath)
	handleError(err)

	data, err := m.Raw()
	handleError(err)

	handleErrors(m.Build(app, "src", cache))
	handleErrors(m.Push(app, registryAddress, buildId, repository))

	b, err := rackClient.UpdateBuild(os.Getenv("APP"), os.Getenv("BUILD"), string(data), "complete", "")
	handleError(err)
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err.Error())

		_, cerr := rackClient.UpdateBuild(os.Getenv("APP"), os.Getenv("BUILD"), "", "failed", err.Error())
		if cerr != nil {
			fmt.Println(cerr.Error())
			os.Exit(2)
		}

		os.Exit(1)
	}
}

func handleErrors(errs []error) {
	for _, err := range errs {
		handleError(err)
	}
}

// buildTar reads a .tgz from stdin, decompresses it, then builds images
func extractTar() {
	cwd, err := os.Getwd()
	handleError(err)
	defer os.Chdir(cwd)

	handleError(os.MkdirAll("src", 0755))
	handleError(os.Chdir("src"))

	cmd := exec.Command("tar", "xz")
	cmd.Stdin = os.Stdin
	handleError(cmd.Run())
}

// buildGitURL takes a URL to a git repo with an optional "commit-ish" hash,
// clones it, checks out the right commit-ish, then builds images
func cloneGit(url string) {

}

func writeDockerAuth() {
	auth := os.Getenv("DOCKER_AUTH")
	handleError(os.MkdirAll("/root/.docker", 0700))
	handleError(ioutil.WriteFile("/root/.docker/config.json", []byte(auth), 0400))
}
