package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strings"

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
		fmt.Println("usage: build <src>")
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

	_, err = rackClient.UpdateBuild(os.Getenv("APP"), os.Getenv("BUILD"), string(data), "complete", "")
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

// extractTar makes a src directory, reads a .tgz from stdin and decompresses it into src
func extractTar() {
	handleError(os.MkdirAll("src", 0755))
	run("src", "tar", "xz")
}

// cloneGit takes a URL to a git repo with an optional "commit-ish" hash,
// clones it, checks out the right commit-ish, and restores original file creation time
func cloneGit(s string) {
	u, err := url.Parse(s)
	handleError(err)

	// if URL has a fragment, i.e. http://github.com/nzoschke/httpd.git#1a2b4aac045609f09de34294de61b45344f419de
	// split it off and pass along http://github.com/nzoschke/httpd.git for `git clone`
	commitish := u.Fragment
	u.Fragment = ""
	repo := u.String()

	// if URL is a ssh/git url, i.e. ssh://user:base64(privatekey)@server/project.git
	// decode and write private key to disk and pass along user@service:project.git for `git clone`
	if u.Scheme == "ssh" {
		repo = fmt.Sprintf("%s@%s%s", u.User.Username(), u.Host, u.Path)

		if pass, ok := u.User.Password(); ok {
			key, err := base64.StdEncoding.DecodeString(pass)
			handleError(err)

			handleError(os.Mkdir("/root/.ssh", 0700))
			handleError(ioutil.WriteFile("/root/.ssh/id_rsa", key, 0400))
		}

		// don't interactive prompt for known hosts and fingerprints
		os.Setenv("GIT_SSH_COMMAND", "ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no")
	}

	writeAsset("/usr/local/bin/git-restore-mtime", "git-restore-mtime", 0755, nil)

	run(".", "git", "clone", "--recursive", "--progress", repo, "src")

	if commitish != "" {
		run("src", "git", "checkout", commitish)
	}

	run("src", "/usr/local/bin/git-restore-mtime", ".")
}

// run optionally changes into a directory then executes the command and args
// connected to the OS stdin/stdout/stderr
func run(dir string, name string, arg ...string) {
	sarg := fmt.Sprintf("%v", arg)
	fmt.Printf("RUNNING: %s %s\n", name, sarg[1:len(sarg)-1])

	// optionally change directory and change back at the end of this func
	if dir != "" || dir != "." {
		cwd, err := os.Getwd()
		handleError(err)
		defer os.Chdir(cwd)
		handleError(os.Chdir(dir))
	}

	cmd := exec.Command(name, arg...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	handleError(cmd.Run())
}

func writeAsset(target, name string, perms os.FileMode, replacements map[string]string) {
	data, err := Asset(fmt.Sprintf("data/%s", name))
	handleError(err)

	sdata := string(data)

	if replacements != nil {
		for key, val := range replacements {
			sdata = strings.Replace(sdata, key, val, -1)
		}
	}

	handleError(ioutil.WriteFile(target, []byte(sdata), perms))
}

func writeDockerAuth() {
	auth := os.Getenv("DOCKER_AUTH")
	if auth != "" {
		handleError(os.MkdirAll("/root/.docker", 0700))
		handleError(ioutil.WriteFile("/root/.docker/config.json", []byte(auth), 0400))
	}
}
