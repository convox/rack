package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/convox/rack/api/manifest"
)

var (
	manifestPath    string
	app             string
	cache           = true
	registryAddress string
	buildId         string
	repository      string
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
	fmt.Printf("ARGS: %+v\n", os.Args)
	fmt.Printf("Environ: %+v", os.Environ())

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

	m, err := manifest.Read("src", manifestPath)
	handleError(err)

	handleErrors(m.Build(app, "src", cache))
	handleErrors(m.Push(app, registryAddress, buildId, repository))

	// report back
	//   manifest
	// success or error
	//
	// or error status / reason

	// c := client.New(os.Getenv("RACK_HOST"), os.Getenv("RACK_PASSWORD"), "build")
	// _, err := c.UpdateBuild(os.Getenv("APP"), os.Getenv("BUILD"), "web:", "complete", "")
	// if err != nil {
	// 	panic(err)
	// }
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err.Error())
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

	cmd := exec.Command("tar", "xzv")
	cmd.Stdin = os.Stdin
	handleError(cmd.Run())
}

// buildGitURL takes a URL to a git repo with an optional "commit-ish" hash,
// clones it, checks out the right commit-ish, then builds images
func cloneGit(url string) {

}
