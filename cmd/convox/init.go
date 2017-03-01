package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/convox/rack/cmd/convox/appinit"
	"github.com/convox/rack/cmd/convox/helpers"
	"github.com/convox/rack/cmd/convox/stdcli"
	"gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "init",
		Description: "initialize an app for local development",
		Usage:       "[directory]",
		Action:      cmdInit,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "boilerplate",
				Usage: "generate a simple boilerplate app",
			},
		},
	})
}

func cmdInit(c *cli.Context) error {
	ep := stdcli.QOSEventProperties{Start: time.Now()}

	distinctID, err := currentId()
	if err != nil {
		stdcli.QOSEventSend("cli-init", distinctID, stdcli.QOSEventProperties{Error: err})
	}

	wd := "."

	if len(c.Args()) > 0 {
		wd = c.Args()[0]
	}

	dir, _, err := stdcli.DirApp(c, wd)
	if err != nil {
		stdcli.QOSEventSend("cli-init", distinctID, stdcli.QOSEventProperties{Error: err})
		return stdcli.Error(err)
	}

	if helpers.Exists(path.Join(dir, "docker-compose.yml")) {
		fmt.Println("found docker-compose.yml, try `convox start` instead")
		return nil
	}

	appType, err := initApplication(dir, c)
	if err != nil {
		stdcli.QOSEventSend("Dev Code Update Failed", distinctID, stdcli.QOSEventProperties{Error: err, AppType: appType})
		stdcli.QOSEventSend("cli-init", distinctID, stdcli.QOSEventProperties{Error: err, AppType: appType})
		return stdcli.Error(err)
	}

	stdcli.QOSEventSend("Dev Code Updated", distinctID, stdcli.QOSEventProperties{AppType: appType})
	stdcli.QOSEventSend("cli-init", distinctID, ep)
	return nil
}

// appKind maps to the various buildpacks and their detect output
var appKind = map[string]string{
	"Clojure (Leiningen 2)": "clojure",
	"Clojure":               "clojure",
	"Go":                    "go",
	"Gradle":                "gradle",
	"Java":                  "java",
	"Node.js":               "nodejs",
	"PHP":                   "php",
	"Python":                "python",
	"Ruby":                  "ruby",
	"Scala":                 "scala",
}

func initApplication(dir string, c *cli.Context) (string, error) {
	prepURL := "https://convox.com/docs/preparing-an-application/"
	args := []string{"run", "--rm", "-v", fmt.Sprintf("%s:/app", dir), "convox/init"}

	stdcli.Spinner.Prefix = "Updating convox/init... "
	stdcli.Spinner.Start()

	if err := updateInit(); err != nil {
		fmt.Printf("\x08\x08FAILED\n")
	} else {
		fmt.Printf("\x08\x08OK\n")
	}
	stdcli.Spinner.Stop()

	var kind = ""
	if c.Bool("boilerplate") || emptyDir(dir) {
		kind = "boilerplate"
	} else {
		k, err := exec.Command(dockerBin, append(args, "detect")...).Output()
		if err != nil {
			return "", fmt.Errorf("unable to detect app type: convox/init - %s", err)
		}

		kd := strings.TrimSpace(string(k))
		var ok bool
		kind, ok = appKind[kd]
		if !ok {
			if kd == "" {
				kd = "?"
			}
			return kind, fmt.Errorf("unknown app type: %s \nCheck out %s for more information", kd, prepURL)
		}
	}

	fmt.Printf("Initializing a %s app\n", kind)

	var af appinit.AppFramework

	switch kind {
	case "ruby":
		af = &appinit.RubyApp{}
	case "boilerplate":
		af = &appinit.Boilerplate{}
	default:
		af = &appinit.SimpleApp{
			Kind: kind,
		}
	}

	stdcli.Spinner.Prefix = "Building app metadata. This could take a while... "
	stdcli.Spinner.Start()
	if err := af.Setup(dir); err != nil {
		fmt.Printf("\x08\x08FAILED\n")
		stdcli.Spinner.Stop()
		return kind, err
	}
	fmt.Printf("\x08\x08OK\n")
	stdcli.Spinner.Stop()

	m, err := af.GenerateManifest()
	if err != nil {
		return kind, err
	}

	if err := writeFile("docker-compose.yml", m, 0644); err != nil {
		return kind, err
	}

	df, err := af.GenerateDockerfile()
	if err != nil {
		return kind, err
	}

	if err := writeFile("Dockerfile", df, 0644); err != nil {
		return kind, err
	}

	di, err := af.GenerateDockerIgnore()
	if err != nil {
		return kind, err
	}

	if err := writeFile(".dockerignore", di, 0644); err != nil {
		return kind, err
	}

	gi, err := af.GenerateGitIgnore()
	if err != nil {
		return kind, err
	}

	if err := writeFile(".gitignore", gi, 0644); err != nil {
		return kind, err
	}

	env, err := af.GenerateLocalEnv()
	if err != nil {
		return kind, err
	}

	if err := writeFile(".env", env, 0644); err != nil {
		return kind, err
	}

	cleanComposeFile()

	fmt.Println()
	fmt.Println("Try running `convox start`")
	return kind, err
}

func updateInit() error {
	cmd := exec.Command("docker", "pull", "convox/init")
	return cmd.Run()
}

// cleanComposeFile removes known invalid fields from a docker-compose.yml file
// due to limitations in the yaml pkg not applying `omitempty` to zero valued structs.
func cleanComposeFile() error {
	file, err := os.Open("docker-compose.yml")
	if err != nil {
		return err
	}
	defer file.Close()

	var buffer bytes.Buffer
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		switch strings.TrimSpace(scanner.Text()) {
		case "build: {}", "command: null":
			continue
		default:
			buffer.WriteString(scanner.Text() + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return ioutil.WriteFile("docker-compose.yml", buffer.Bytes(), 0644)
}

// emptyDir checks if the directory is empty
func emptyDir(dir string) bool {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return false
	}

	if len(files) == 0 {
		return true
	}

	return false
}

// writeFile is a helper function that writes a file
func writeFile(path string, data []byte, mode os.FileMode) error {
	if data == nil {
		return nil
	}

	fmt.Printf("Writing %s... ", path)

	if helpers.Exists(path) {
		fmt.Println("EXISTS")
		return nil
	}

	// make the containing directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(path, data, mode); err != nil {
		return err
	}

	fmt.Println("OK")
	return nil
}
