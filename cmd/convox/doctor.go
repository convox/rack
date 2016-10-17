package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/manifest"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/equinox-io/equinox"
	docker "github.com/fsouza/go-dockerclient"
	cli "gopkg.in/urfave/cli.v1"
)

func init() {
	stdcli.RegisterCommand(cli.Command{
		Name:        "doctor",
		Action:      cmdDoctor,
		Description: "Check your app for common Convox compatibility issues.",
	})
}

type Diagnosis struct {
	Kind        string
	Title       string
	Description string
	DocsLink    string
}

func (d Diagnosis) String() string {
	var icon string
	var link string

	switch d.Kind {
	case "success":
		icon = "[<success>\u2713</success>]"
	case "warning":
		icon = "[<warning>!</warning>]"
	case "fail":
		icon = "[<fail>X</fail>]"
	default:
		icon = "[<warning>?</warning>]"
	}

	body := ""
	if d.Description != "" {
		body = fmt.Sprintf("%s\n", d.Description)
	}

	if d.DocsLink != "" {
		link = fmt.Sprintf("<link>%s</link>\n", d.DocsLink)
	}
	return fmt.Sprintf("%s %s    \n%s%s", icon, d.Title, body, link)
}

var (
	diagnoses  = []Diagnosis{}
	docContext = &cli.Context{}

	setupChecks = []func() error{
		checkCLIVersion,
		checkDockerRunning,
		checkDockerVersion,
		checkDockerPull,
	}

	buildImageChecks = []func() error{
		checkDockerfile,
		checkDockerignoreGit,
		checkLargeFiles,
		checkBuildDocker,
	}

	buildServiceChecks = []func(*manifest.Manifest) error{
		checkVersion2,
		checkMissingDockerFiles,
		checkValidServices,
	}

	buildEnvironmentChecks = []func(*manifest.Manifest) error{
		checkEnvFound,
		checkEnvValid,
		checkEnvIgnored,
		checkMissingEnv,
	}

	runBalancerChecks = []func(*manifest.Manifest) error{
		checkAppExposesPorts,
	}

	runResourceChecks = []func(*manifest.Manifest) error{
		checkAppDefinesResource,
		checkValidResources,
	}

	runLinkChecks = []func(*manifest.Manifest) error{
		checkAppDefinesLink,
		checkValidLinks,
	}
)

func startCheck(title string) {
	stdcli.Spinner.Prefix = fmt.Sprintf("[ ] %s ", stdcli.Sprintf(title))
	stdcli.Spinner.Start()
}

func diagnose(d Diagnosis) {
	stdcli.Spinner.Stop()
	time.Sleep(100 * time.Millisecond)
	print("\033[K")

	stdcli.Writef(d.String())
	if d.Kind == "fail" {
		os.Exit(1)
	}
}

func cmdDoctor(c *cli.Context) error {
	docContext = c
	stdcli.Writef("### Setup\n")
	for _, check := range setupChecks {
		if err := check(); err != nil {
			return stdcli.Error(err)
		}
	}

	stdcli.Writef("\n\n### Build: Image\n")
	for _, check := range buildImageChecks {
		if err := check(); err != nil {
			return stdcli.Error(err)
		}
	}

	stdcli.Writef("\n\n### Build: Service\n")
	startCheck("<file>docker-compose.yml</file> found")
	_, err := os.Stat("docker-compose.yml")
	if err != nil {
		diagnose(Diagnosis{
			Title:       "<file>docker-compose.yml</file> found",
			Description: "<fail>A docker-compose.yml file is required to define Services</fail>",
			Kind:        "fail",
			DocsLink:    "https://convox.com/guide/service/",
		})
	} else {
		diagnose(Diagnosis{
			Title: "<file>docker-compose.yml</file> found",
			Kind:  "success",
		})
	}

	m, err := manifest.LoadFile("docker-compose.yml")
	checkManifestValid(m, err)
	for _, check := range buildServiceChecks {
		if err := check(m); err != nil {
			return stdcli.Error(err)
		}
	}

	stdcli.Writef("\n\n### Build: Environment\n")
	for _, check := range buildEnvironmentChecks {
		if err := check(m); err != nil {
			return stdcli.Error(err)
		}
	}

	stdcli.Writef("\n\n### Run: Balancer\n")
	for _, check := range runBalancerChecks {
		if err := check(m); err != nil {
			return stdcli.Error(err)
		}
	}

	stdcli.Writef("\n\n### Run: Resource\n")
	for _, check := range runResourceChecks {
		if err := check(m); err != nil {
			return stdcli.Error(err)
		}
	}

	stdcli.Writef("\n\n### Run: Link\n")
	for _, check := range runLinkChecks {
		if err := check(m); err != nil {
			return stdcli.Error(err)
		}
	}

	stdcli.Writef("\n\n<success>Success:</success> Your app looks ready for development. \nRun it with `convox start`.\n\n")
	return nil
}

func checkDockerRunning() error {
	startCheck("Docker running")

	dockerTest := exec.Command("docker", "images")
	err := dockerTest.Run()
	if err != nil {
		diagnose(Diagnosis{
			Title:       "Docker running",
			Description: "<fail>Could not connect to the Docker daemon, is it installed and running?</fail>",
			DocsLink:    "https://docs.docker.com/engine/installation/",
			Kind:        "fail",
		})
		return nil
	} else {
		diagnose(Diagnosis{
			Title: "Docker running",
			Kind:  "success",
		})
	}
	return nil
}

func checkDockerVersion() error {
	startCheck("Docker up to date")
	dockerVersionTest, err := docker.NewClientFromEnv()
	if err != nil {
		return err
	}

	minDockerVersion, err := docker.NewAPIVersion("1.9")
	e, err := dockerVersionTest.Version()
	if err != nil {
		return err
	}

	currentVersionParts := strings.Split(e.Get("Version"), ".")
	currentVersion, err := docker.NewAPIVersion(fmt.Sprintf("%s.%s", currentVersionParts[0], currentVersionParts[1]))
	if err != nil {
		return err
	}

	if !(currentVersion.GreaterThanOrEqualTo(minDockerVersion)) {
		diagnose(Diagnosis{
			Title:       "Docker up to date",
			Description: "<fail>Docker engine is out of date (min: 1.9)</fail>",
			DocsLink:    "https://docs.docker.com/engine/installation/",
			Kind:        "fail",
		})
	} else {
		diagnose(Diagnosis{
			Title: "Docker up to date",
			Kind:  "success",
		})
	}
	return nil
}

func checkDockerPull() error {
	title := "Docker pull hello-world works"
	startCheck(title)

	dockerTest := exec.Command("docker", "pull", "hello-world")
	err := dockerTest.Run()
	if err != nil {
		diagnose(Diagnosis{
			Title:       title,
			Description: "<fail>Could not pull and the hello-world Image, is your internet connection ok?</fail>",
			DocsLink:    "https://convox.com/docs/troubleshooting-docker/",
			Kind:        "fail",
		})
		return nil
	} else {
		diagnose(Diagnosis{
			Title: title,
			Kind:  "success",
		})
	}
	return nil
}

func checkCLIVersion() error {
	title := "Convox CLI version"
	startCheck(title)

	client, err := updateClient()
	if err != nil {
		return stdcli.Error(err)
	}

	opts := equinox.Options{
		CurrentVersion: Version,
		Channel:        "stable",
		HTTPClient:     client,
	}
	if err := opts.SetPublicKeyPEM(publicKey); err != nil {
		return stdcli.Error(err)
	}

	// check for update
	_, err = equinox.Check("app_i8m2L26DxKL", opts)
	if err == nil && Version != "dev" {
		diagnose(Diagnosis{
			Kind:        "warning",
			Title:       title,
			Description: "<warning>Your Convox CLI is out of date, run `convox update`</warning>",
		})
	} else {
		diagnose(Diagnosis{
			Title: title,
			Kind:  "success",
		})
	}
	return nil
}

func checkDockerfile() error {
	if df := filepath.Join(filepath.Dir(os.Args[0]), "docker-compose.yml"); exists(df) {
		m, err := manifest.LoadFile("docker-compose.yml")
		if err != nil {
			//This will get picked up later in the test suite
			return nil
		}
		checkMissingDockerFiles(m)
		return nil
	}

	title := "Dockerfile found"
	startCheck(title)

	//Skip if docker-compose file exists
	_, err := os.Stat("docker-compose.yml")
	if err == nil {
		return nil
	}

	_, err = os.Stat("Dockerfile")
	if err != nil {
		diagnose(Diagnosis{
			Title:       title,
			Description: "<fail>A Dockerfile is required to build an Image</fail>",
			Kind:        "fail",
			DocsLink:    "https://convox.com/guide/build/",
		})
	} else {
		diagnose(Diagnosis{
			Title: title,
			Kind:  "success",
		})
	}
	return nil
}

func checkDockerfileValid() error {
	//TODO
	return nil
}

func checkDockerignoreGit() error {
	title := "<file>.git</file> in <file>.dockerignore</file>"
	startCheck(title)

	_, err := os.Stat(".dockerignore")
	if err != nil {
		diagnose(Diagnosis{
			Title:       title,
			Description: "<warning>It looks like you don't have a .dockerignore file</warning>",
			Kind:        "warning",
			DocsLink:    "https://docs.docker.com/engine/reference/builder/#/dockerignore-file",
		})
		return nil
	}

	// read the whole file at once
	b, err := ioutil.ReadFile(".dockerignore")
	if err != nil {
		return err
	}
	s := string(b)

	// //check whether s contains substring text
	if !strings.Contains(s, ".git\n") {
		diagnose(Diagnosis{
			Title:       title,
			Kind:        "warning",
			DocsLink:    "https://docs.docker.com/engine/reference/builder/#/dockerignore-file",
			Description: "<warning>You should probably add .git to your .dockerignore</warning>",
		})
		return nil
	}

	diagnose(Diagnosis{
		Title: title,
		Kind:  "success",
	})

	return nil
}

func checkLargeFiles() error {
	title := "Large files in <file>.dockerignore</file>"
	startCheck(title)

	files := map[string]int64{}
	message := ""

	di, _ := readDockerIgnore(".")

	f := func(path string, info os.FileInfo, err error) error {
		m, err := fileutils.Matches(path, di)
		if err != nil {
			return err
		}
		if m {
			return nil
		}
		if info.Size() >= 200000000 {
			files[path] = info.Size()
		}
		return nil
	}

	err := filepath.Walk(".", f)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		diagnose(Diagnosis{
			Title: title,
			Kind:  "success",
		})
		return nil
	}

	for k, v := range files {
		message += fmt.Sprintf(
			"<warning>./%s is %d bytes, perhaps you should add it to your .dockerignore to speed up builds and deploys</warning>\n",
			k,
			v,
		)
	}

	diagnose(Diagnosis{
		Title:       title,
		Kind:        "warning",
		DocsLink:    "https://docs.docker.com/engine/reference/builder/#/dockerignore-file",
		Description: message,
	})
	return nil
}

func checkBuildDocker() error {
	title := "Image builds successfully"

	if df := filepath.Join(filepath.Dir(os.Args[0]), "docker-compose.yml"); exists(df) {
		m, err := manifest.LoadFile(df)
		if err != nil {
			//This will be handled later in the suite
			return nil
		}

		startCheck(title)

		_, app, err := stdcli.DirApp(docContext, ".")
		if err != nil {
			//This will be handled later in the suite
			return nil
		}

		s := make(chan string)
		output := []string{}

		go func() {
			for x := range s {
				output = append(output, x)
			}
		}()
		err = m.Build(".", app, s, true)

		if err != nil {
			message := ""
			for _, x := range output {
				message += fmt.Sprintf("<fail>%s</fail>\n", x)
			}
			diagnose(Diagnosis{
				Title:       title,
				Description: message,
				Kind:        "fail",
			})
		}

		diagnose(Diagnosis{
			Title: title,
			Kind:  "success",
		})
		return nil
	}

	startCheck(title)

	byts, err := exec.Command("docker", "build", ".").CombinedOutput()
	if err != nil {
		bytsArr := strings.Split(string(byts), "\n")
		message := ""
		for _, x := range bytsArr {
			message += fmt.Sprintf("<description>%s</description>\n", x)
		}
		diagnose(Diagnosis{
			Title:       title,
			Description: message,
			Kind:        "fail",
		})
		return nil
	}

	diagnose(Diagnosis{
		Title: title,
		Kind:  "success",
	})
	return nil
}

func checkManifestValid(m *manifest.Manifest, parseError error) error {
	title := "<file>docker-compose.yml</file> valid"
	startCheck(title)

	if parseError != nil {
		diagnose(Diagnosis{
			Title:       title,
			Kind:        "fail",
			DocsLink:    "https://convox.com/docs/docker-compose-file/",
			Description: "<description>docker-compose.yml is not valid YAML</description>",
		})
		return nil
	}

	errs := m.Validate()
	if len(errs) > 0 {
		body := ""
		for _, err := range errs {
			body += fmt.Sprintf("<description>%s</description>\n", err.Error())
		}
		diagnose(Diagnosis{
			Title:       title,
			Kind:        "fail",
			DocsLink:    "https://convox.com/docs/docker-compose-file/",
			Description: body,
		})
		return nil
	}

	diagnose(Diagnosis{
		Kind:  "success",
		Title: title,
	})
	return nil
}

func checkVersion2(m *manifest.Manifest) error {
	title := "<file>docker-compose.yml</file> version 2"
	startCheck(title)
	if m.Version == "2" {
		diagnose(Diagnosis{
			Kind:  "success",
			Title: title,
		})
		return nil
	}
	diagnose(Diagnosis{
		Title:       title,
		Kind:        "warning",
		DocsLink:    "https://convox.com/docs/docker-compose-file/",
		Description: "<warning>You are using the legacy v1 docker-compose.yml</warning>",
	})
	return nil
}

func checkEnvFound(m *manifest.Manifest) error {
	title := "<file>.env</file> found"
	startCheck(title)

	_, err := os.Stat(".env")
	if err != nil {
		diagnose(Diagnosis{
			Title:       title,
			Description: "<warning>A .env file is recommended to manage development configuration</warning>",
			Kind:        "warning",
			DocsLink:    "https://convox.com/guide/environment/",
		})
	} else {
		diagnose(Diagnosis{
			Title: title,
			Kind:  "success",
		})
	}
	return nil
}

func checkEnvValid(m *manifest.Manifest) error {
	//TODO
	if denv := filepath.Join(filepath.Dir(os.Args[0]), ".env"); exists(denv) {
	}
	return nil
}

func checkEnvIgnored(m *manifest.Manifest) error {
	//TODO
	if denv := filepath.Join(filepath.Dir(os.Args[0]), ".env"); exists(denv) {
		title := "<file>.env</file> in <file>.gitignore</file> and <file>.dockerignore</file>"
		startCheck(title)
		_, err := os.Stat(".dockerignore")
		if err != nil {
			diagnose(Diagnosis{
				Title:       title,
				Description: "<warning>It looks like you don't have a .dockerignore file</warning>",
				Kind:        "warning",
				DocsLink:    "https://docs.docker.com/engine/reference/builder/#/dockerignore-file",
			})
			return nil
		}

		_, err = os.Stat(".gitignore")
		if err != nil {
			diagnose(Diagnosis{
				Title:       title,
				Description: "<warning>It looks like you don't have a .gitignore file</warning>",
				Kind:        "warning",
				DocsLink:    "https://git-scm.com/docs/gitignore",
			})
			return nil
		}

		dockerIgnore, err := ioutil.ReadFile(filepath.Join(filepath.Dir(os.Args[0]), ".dockerignore"))
		if err != nil {
			return err
		}

		gitIgnore, err := ioutil.ReadFile(filepath.Join(filepath.Dir(os.Args[0]), ".gitignore"))
		if err != nil {
			return err
		}

		dockerIgnoreLines := strings.Split(string(dockerIgnore), "\n")
		gitIgnoreLines := strings.Split(string(gitIgnore), "\n")

		dockerIgnored, _ := fileutils.Matches(".env", dockerIgnoreLines)
		gitIgnored, _ := fileutils.Matches(".env", gitIgnoreLines)

		if !gitIgnored {
			diagnose(Diagnosis{
				Title:       title,
				Description: "<warning>It looks like you don't have your .env in your .gitignore file</warning>",
				Kind:        "warning",
				DocsLink:    "https://git-scm.com/docs/gitignore",
			})
		}

		if !dockerIgnored {
			diagnose(Diagnosis{
				Title:       title,
				Description: "<warning>It looks like you don't have your .env in your .dockerignore file</warning>",
				Kind:        "warning",
				DocsLink:    "https://docs.docker.com/engine/reference/builder/#/dockerignore-file",
			})
		}
		diagnose(Diagnosis{
			Title: title,
			Kind:  "success",
		})
	}
	return nil
}

func checkMissingEnv(m *manifest.Manifest) error {
	if denv := filepath.Join(filepath.Dir(os.Args[0]), ".env"); exists(denv) {
		data, err := ioutil.ReadFile(denv)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "=") {
				parts := strings.SplitN(scanner.Text(), "=", 2)

				err := os.Setenv(parts[0], parts[1])
				if err != nil {
					return err
				}
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}
	} else {
		return nil
	}

	// check for required env vars
	existing := map[string]bool{}
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			existing[parts[0]] = true
		}
	}

	for _, s := range manifestServices(m) {
		title := fmt.Sprintf("Service <service>%s</service> <config>environment</config> found in <file>.env</file>", s.Name)
		startCheck(title)

		links := map[string]bool{}

		for _, l := range s.Links {
			prefix := strings.ToUpper(l) + "_"
			prefix = strings.Replace(prefix, "-", "_", -1)

			key := prefix + "URL"
			links[key] = true
		}

		missingEnv := []string{}
		for key, val := range s.Environment {
			eok := val != ""
			_, exok := existing[key]
			_, lok := links[key]
			if !eok && !exok && !lok {
				missingEnv = append(missingEnv, key)
			}
		}

		if len(missingEnv) > 0 {
			diagnose(Diagnosis{
				Title:       title,
				Kind:        "fail",
				DocsLink:    "https://convox.com/guide/environment/",
				Description: fmt.Sprintf("<fail>development environment var not set: %s</fail>", strings.Join(missingEnv, ", ")),
			})
		}
		diagnose(Diagnosis{
			Title: title,
			Kind:  "success",
		})

	}

	return nil
}

// func syncVolumeConflict(m *manifest.Manifest) error {
// 	for _, s := range m.Services {
// 		sps, err := s.SyncPaths()
// 		if err != nil {
// 			return err
// 		}

// 		for _, v := range s.Volumes {
// 			parts := strings.Split(v, ":")
// 			if len(parts) == 2 {
// 				for k, _ := range sps {
// 					if k == parts[0] {
// 						// diagnose(Diagnosis{
// 						// 	Kind:     "warning",
// 						// 	DocsLink: "#TODO",
// 						// 	Description: fmt.Sprintf(
// 						// 		"<description>service: %s has a sync path conflict with volume %s</description>",
// 						// 		s.Name,
// 						// 		v),
// 						// })
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return nil
// }

func checkMissingDockerFiles(m *manifest.Manifest) error {
	title := "Dockerfiles found"
	startCheck(title)

	for _, s := range m.Services {
		if s.Image == "" {
			dockerFile := coalesce(s.Dockerfile, "Dockerfile")
			dockerFile = coalesce(s.Build.Dockerfile, dockerFile)
			_, err := os.Stat(fmt.Sprintf("%s/%s", s.Build.Context, dockerFile))
			if err != nil {
				diagnose(Diagnosis{
					Title:       title,
					Kind:        "fail",
					DocsLink:    "https://convox.com/guide/image/",
					Description: fmt.Sprintf("<fail>Service <service>%s</service> is missing a Dockerfile</fail>", s.Name),
				})
			}
		}
	}
	diagnose(Diagnosis{
		Title: title,
		Kind:  "success",
	})
	return nil
}

func checkValidServices(m *manifest.Manifest) error {
	_, app, err := stdcli.DirApp(docContext, ".")
	if err != nil {
		return err
	}
	for _, s := range manifestServices(m) {
		title := fmt.Sprintf("Service <service>%s</service> is valid", s.Name)
		startCheck(title)
		if s.Command.String != "" || (s.Command.Array != nil && len(s.Command.Array) == 0) {
			diagnose(Diagnosis{
				Title: title,
				Kind:  "success",
			})
			continue
		}

		t := s.Tag(app)
		dockerCli, err := docker.NewClientFromEnv()
		if err != nil {
			return err
		}

		i, err := dockerCli.InspectImage(t)
		if err != nil {
			return err
		}

		if (len(i.Config.Cmd) > 0) || (i.Config.Entrypoint != nil) {
			diagnose(Diagnosis{
				Title: title,
				Kind:  "success",
			})
			continue
		}
		diagnose(Diagnosis{
			Title:       title,
			Kind:        "fail",
			DocsLink:    "http://convox.com/guide/service/",
			Description: fmt.Sprintf("<fail>Service <service>%s</service> doesn't have a valid command</fail>", s.Name),
		})
	}
	return nil
}

func checkAppExposesPorts(m *manifest.Manifest) error {
	title := "App exposes ports"
	startCheck(title)

	for _, s := range m.Services {
		if len(s.Ports) > 0 {
			diagnose(Diagnosis{
				Title: title,
				Kind:  "success",
			})
			return nil
		}
	}
	diagnose(Diagnosis{
		Title:       title,
		Kind:        "warning",
		DocsLink:    "http://convox.com/guide/balancer/",
		Description: "<warning>This app does not expose any ports</warning>",
	})
	return nil
}

func checkAppDefinesResource(m *manifest.Manifest) error {
	title := "App defines Resources"
	startCheck(title)

	if len(manifestResources(m)) > 0 {
		diagnose(Diagnosis{
			Title: title,
			Kind:  "success",
		})
		return nil
	}

	diagnose(Diagnosis{
		Title:       title,
		Kind:        "warning",
		DocsLink:    "http://convox.com/guide/resource/",
		Description: "<warning>This app does not define any Resources</warning>",
	})
	return nil
}

func checkValidResources(m *manifest.Manifest) error {
	rs := manifestResources(m)

	if len(rs) == 0 {
		return nil
	}

	for _, s := range rs {
		title := fmt.Sprintf("Resource <resource>%s</resource> is valid", s.Name)
		startCheck(title)

		diagnose(Diagnosis{
			Title: title,
			Kind:  "success",
		})
	}

	return nil
}

func manifestServices(m *manifest.Manifest) []manifest.Service {
	services := []manifest.Service{}

	resources := manifestResources(m)
	resourceNames := map[string]bool{}

	for _, r := range resources {
		resourceNames[r.Name] = true
	}

	for _, s := range m.Services {
		if _, ok := resourceNames[s.Name]; ok {
			continue
		}
		services = append(services, s)
	}

	return services
}

func manifestResources(m *manifest.Manifest) []manifest.Service {
	resources := []manifest.Service{}

	for _, s := range m.Services {
		prebuiltImage := strings.HasPrefix(s.Image, "convox/")
		noCommand := s.Command.String == "" && s.Command.Array == nil
		if prebuiltImage && noCommand {
			resources = append(resources, s)
		}
	}

	return resources
}

func checkAppDefinesLink(m *manifest.Manifest) error {
	title := "App defines Links"
	startCheck(title)

	for _, s := range m.Services {
		if len(s.Links) > 0 {
			diagnose(Diagnosis{
				Title: title,
				Kind:  "success",
			})
			return nil
		}
	}

	diagnose(Diagnosis{
		Title:       title,
		Kind:        "warning",
		DocsLink:    "http://convox.com/guide/link/",
		Description: "<warning>This app does not define any Links</warning>",
	})
	return nil
}

func checkValidLinks(m *manifest.Manifest) error {
	resourceNames := map[string]bool{}

	for _, s := range manifestServices(m) {
		linkVars := []string{}
		missingEnv := []string{}

		for _, l := range s.Links {
			resourceNames[l] = true

			prefix := strings.ToUpper(l) + "_"
			prefix = strings.Replace(prefix, "-", "_", -1)

			key := prefix + "URL"
			linkVars = append(linkVars, key)

			missing := true
			for k, _ := range s.Environment {
				if k == key {
					missing = false
					break
				}
			}

			if missing {
				missingEnv = append(missingEnv, key)
			}
		}

		title := fmt.Sprintf("Service <service>%s</service> environment expects %s", s.Name, strings.Join(linkVars, ", "))

		if len(missingEnv) > 0 {
			diagnose(Diagnosis{
				Title:       title,
				Kind:        "fail",
				DocsLink:    "https://convox.com/guide/link/",
				Description: fmt.Sprintf("<fail>Service <service>%s</service> not expecting %s</fail>", s.Name, strings.Join(missingEnv, ", ")),
			})
		}
		diagnose(Diagnosis{
			Title: title,
			Kind:  "success",
		})
	}

	resources := manifestResources(m)

	for _, r := range resources {
		title := fmt.Sprintf("Resource <resource>%s</resource> exposes internal port", r.Name)

		if _, ok := resourceNames[r.Name]; ok {
			if len(r.InternalPorts()) == 0 {
				diagnose(Diagnosis{
					Title:       title,
					Kind:        "error",
					DocsLink:    "http://convox.com/guide/link/",
					Description: fmt.Sprintf("<warning>Resource <resource>%s</resource> does not expose an internal port</warning>", r.Name),
				})
			} else {
				diagnose(Diagnosis{
					Title: title,
					Kind:  "success",
				})
			}
		}
	}

	return nil
}

// func checkUnsupportedFeatures(m *manifest.Manifest) error {
// 	dc, err := ioutil.ReadFile("docker-compose.yml")
// 	if err != nil {
// 		return err
// 	}

// 	r, err := m.Raw()
// 	if err != nil {
// 		return err
// 	}

// 	dcLines := strings.Split(string(dc), "\n")
// 	rLines := strings.Split(string(r), "\n")

// 	adds := 0
// 	discoveries := make([][]int, 0)
// 	current := make([]int, 0)
// 	removes := 0

// 	d := difflib.Diff(rLines, dcLines)

// 	for x, dr := range d {
// 		switch dr.Delta {
// 		case 0:
// 			if len(current) > 0 {
// 				discoveries = append(discoveries, []int{current[0], len(current)})
// 				current = []int{}
// 			}
// 			removes = 0
// 		case 1:
// 			adds++
// 			removes++
// 		case 2:
// 			if removes == 0 {
// 				current = append(current, x-adds)
// 			} else {
// 				removes--
// 			}
// 		}
// 	}

// 	for _, d := range discoveries {
// 		parts := []string{}

// 		head := dcLines[:d[0]]
// 		tail := dcLines[d[0]+d[1]:]
// 		max := len(dcLines)

// 		switch {
// 		case len(head) == 1:
// 			num := d[0] - 1
// 			parts = append(parts, numberedLine(fmt.Sprintf("<description>%s</description>", dcLines[num]), num, max))
// 		case len(head) >= 2:
// 			num1 := d[0] - 2
// 			num2 := d[0] - 1
// 			parts = append(parts, numberedLine(fmt.Sprintf("<description>%s</description>", dcLines[num1]), num1, max))
// 			parts = append(parts, numberedLine(fmt.Sprintf("<description>%s</description>", dcLines[num2]), num2, max))
// 		}

// 		for x := d[0]; x < (d[0] + d[1]); x++ {
// 			parts = append(parts, numberedLine(fmt.Sprintf("<warning>%s</warning>", dcLines[x]), x, max))
// 		}

// 		switch {
// 		case len(tail) == 1:
// 			parts = append(parts, numberedLine(fmt.Sprintf("<description>%s</description>", tail[0]), d[0]+d[1], max))
// 		case len(tail) >= 2:
// 			parts = append(parts, numberedLine(fmt.Sprintf("<description>%s</description>", tail[0]), d[0]+d[1], max))
// 			parts = append(parts, numberedLine(fmt.Sprintf("<description>%s</description>", tail[1]), d[0]+d[1]+1, max))
// 		}

// 		// diagnose(Diagnosis{
// 		// 	Title:       "It looks like you are using docker-compose features that convox doesn't support",
// 		// 	Kind:        "warning",
// 		// 	Description: strings.Join(parts, "\n"),
// 		// 	DocsLink:    "#TODO",
// 		// })
// 	}

// 	return nil
// }

// func numberedLine(line string, num, maxNum int) string {
// 	b := len(fmt.Sprintf("%d", maxNum))
// 	format := fmt.Sprintf("<linenumber>%%0%dd:</linenumber> %%s", b)
// 	return fmt.Sprintf(format, num+1, line)
// }
