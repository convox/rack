package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/convox/rack/cmd/convox/stdcli"
	"github.com/convox/rack/manifest"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/equinox-io/equinox"
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
	DocsLink    string
	Kind        string
	Description string
}

func (d Diagnosis) String() string {
	s := ""
	if d.Kind == "warning" {
		s += "<warning>Warning:</warning> "
	} else if d.Kind == "security" {
		s += "<security>Security:</security> "
	} else {
		s += "<warning>Unknown:</warning> "
	}

	s += d.Description
	s += "\n"
	s += d.DocsLink
	s += "\n\n"
	return s
}

var (
	diagnoses = []Diagnosis{}

	buildChecks = []func(*manifest.Manifest) error{
		// checkCLIVersion,
		checkMissingDockerFiles,
		checkDockerIgnore,
		checkLargeFiles,
	}

	devChecks = []func(*manifest.Manifest) error{
		syncVolumeConflict,
		missingEnvValues,
	}

	// prodChecks  = []func(*manifest.Manifest) error{}

	// manifestChecks = []func(*manifest.Manifest) error{
	// 	validateManifest,
	// }

	// dockerChecks = []func() error{
	// 	dockerTest,
	// }
)

func diagnose(d Diagnosis) {
	diagnoses = append(diagnoses, d)
}

func medicalReport() {
	if len(diagnoses) > 0 {
		stdcli.Writef("\n\n")
		for _, d := range diagnoses {
			stdcli.Writef(d.String())
		}
		os.Exit(1)
	}
}

func cmdDoctor(c *cli.Context) error {
	stdcli.Writef("Running build tests: ")
	m, err := manifest.LoadFile("docker-compose.yml")
	if err != nil {
		return stdcli.Error(err)
	}

	for _, check := range buildChecks {
		if err := check(m); err != nil {
			return stdcli.Error(err)
		}
	}

	medicalReport()
	stdcli.Writef("<success>\u2713</success>\n\n")

	stdcli.Writef("Running development tests: ")
	for _, check := range devChecks {
		if err := check(m); err != nil {
			return stdcli.Error(err)
		}
	}

	medicalReport()
	stdcli.Writef("<success>\u2713</success>\n\n")

	// for _, check := range dockerChecks {
	// 	if err := check(); err != nil {
	// 		return stdcli.Error(err)
	// 	}
	// }

	stdcli.Writef("<success>Success:</success> Your app looks ready for deployment into convox. \nHead to https://console.convox.com to get started\n")
	return nil
}

func checkDockerIgnore(m *manifest.Manifest) error {
	_, err := os.Stat(".dockerignore")
	if err != nil {
		diagnose(Diagnosis{
			Kind:        "security",
			DocsLink:    "#TODO",
			Description: "You should probably have a .dockerignore file",
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
			Kind:        "security",
			DocsLink:    "#TODO",
			Description: "You should probably add .git to your .dockerignore",
		})
	}

	if !strings.Contains(s, ".env\n") {
		diagnose(Diagnosis{
			Kind:        "security",
			DocsLink:    "#TODO",
			Description: "You should probably add .env to your .dockerignore",
		})
	}

	return nil
}

func checkCLIVersion(m *manifest.Manifest) error {
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
	if err == nil {
		diagnose(Diagnosis{
			Kind:        "warning",
			DocsLink:    "#TODO",
			Description: "Your client is out of date, run `convox update`",
		})
	}
	return nil
}

func validateManifest(m *manifest.Manifest) error {
	return m.Validate()
}

func missingEnvValues(m *manifest.Manifest) error {
	_, err := os.Stat(".env")
	if err != nil {
		diagnose(Diagnosis{
			Kind:        "warning",
			DocsLink:    "#TODO",
			Description: "It looks like you are missing a .env file",
		})
		return nil
	}

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
	}

	// check for required env vars
	existing := map[string]bool{}
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			existing[parts[0]] = true
		}
	}

	for _, s := range m.Services {
		links := map[string]bool{}

		for _, l := range s.Links {
			key := fmt.Sprintf("%s_URL", strings.ToUpper(l))
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
				Kind:        "warning",
				DocsLink:    "#TODO",
				Description: fmt.Sprintf("env expected: %s", strings.Join(missingEnv, ", ")),
			})
		}
	}

	return nil
}

func syncVolumeConflict(m *manifest.Manifest) error {
	for _, s := range m.Services {
		sps, err := s.SyncPaths()
		if err != nil {
			return err
		}

		for _, v := range s.Volumes {
			parts := strings.Split(v, ":")
			if len(parts) == 2 {
				for k, _ := range sps {
					if k == parts[0] {
						diagnose(Diagnosis{
							Kind:     "warning",
							DocsLink: "#TODO",
							Description: fmt.Sprintf(
								"service: %s has a sync path conflict with volume %s",
								s.Name,
								v),
						})
					}
				}
			}
		}
	}
	return nil
}

func checkMissingDockerFiles(m *manifest.Manifest) error {
	for _, s := range m.Services {
		if s.Image == "" {
			dockerFile := coalesce(s.Dockerfile, "Dockerfile")
			dockerFile = coalesce(s.Build.Dockerfile, dockerFile)
			_, err := os.Stat(fmt.Sprintf("%s/%s", s.Build.Context, dockerFile))
			if err != nil {
				diagnose(Diagnosis{
					Kind:        "warning",
					DocsLink:    "#TODO",
					Description: fmt.Sprintf("service: %s is missing it's Dockerfile", s.Name),
				})
			}
		}
	}
	return nil
}

func checkLargeFiles(m *manifest.Manifest) error {
	di, err := readDockerIgnore(".")
	if err != nil {
		return err
	}

	f := func(path string, info os.FileInfo, err error) error {
		m, err := fileutils.Matches(path, di)
		if err != nil {
			return err
		}
		if m {
			return nil
		}
		if info.Size() >= 200000000 {
			diagnose(Diagnosis{
				Kind:        "warning",
				DocsLink:    "#TODO",
				Description: fmt.Sprintf("%s is %d, perhaps you should add it to your .dockerignore to speed up builds and deploys", info.Name(), info.Size()),
			})
		}
		return nil
	}

	err = filepath.Walk(".", f)
	if err != nil {
		return err
	}
	return nil
}
