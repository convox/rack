package provider

import (
	"fmt"
	"strings"

	"github.com/convox/rack/composure/structs"
	"github.com/stretchr/testify/mock"
)

var TestProvider = &TestProviderRunner{}

type TestProviderRunner struct {
	mock.Mock
}

func (p *TestProviderRunner) ImageBuild(path, dockerfile, tag string) error {
	p.Called(path, dockerfile, tag)
	return nil
}

func (p *TestProviderRunner) ImageInspect(tag string) (map[string]string, error) {
	args := p.Called(tag)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (p *TestProviderRunner) ImagePull(name string) error {
	p.Called(name)
	return nil
}

func (p *TestProviderRunner) ImagePush(name, url string) error {
	p.Called(name, url)
	return nil
}

func (p *TestProviderRunner) ImageTag(name, tag string) error {
	p.Called(name, tag)
	return nil
}

func (p *TestProviderRunner) NetworkInspect() (string, error) {
	args := p.Called()
	return args.String(0), args.Error(1)
}

func (p *TestProviderRunner) ManifestBuild(path, manifestfile string) (map[string]string, error) {
	p.Called(path, manifestfile)

	nameTags := map[string]string{}

	projectName, err := p.ProjectName(path)
	if err != nil {
		return nameTags, err
	}

	m, err := p.ManifestLoad(path, manifestfile)
	if err != nil {
		return nameTags, err
	}

	// pull, build and tag images
	for serviceName, entry := range *m {
		tag := fmt.Sprintf("%s/%s", projectName, serviceName)
		nameTags[serviceName] = tag

		if entry.Image != "" {
			p.ImagePull(entry.Image)
			p.ImageTag(entry.Image, tag)
		} else if entry.Build != "" {
			df := entry.Dockerfile
			if df == "" {
				df = "Dockerfile"
			}

			tmpTag := randomString("convox-", 10)
			p.ImageBuild(path, df, tmpTag)
			p.ImageTag(tmpTag, tag)
		}
	}

	return nameTags, nil
}

func (p *TestProviderRunner) ManifestLoad(path, manifestfile string) (*structs.Manifest, error) {
	args := p.Called(path, manifestfile)
	return args.Get(0).(*structs.Manifest), args.Error(1)
}

func (p *TestProviderRunner) ManifestPush(path, manifestfile, registry, repository string) error {
	args := p.Called(path, manifestfile, registry, repository)

	nameTags, err := p.ManifestBuild(path, manifestfile)
	if err != nil {
		return err
	}

	for k, v := range nameTags {
		err := p.ImagePush(v, fmt.Sprintf("%s/%s:%s", registry, repository, k))
		if err != nil {
			return err
		}
	}

	return args.Error(0)
}

func (p *TestProviderRunner) ManifestRun(path, manifestfile string) error {
	p.Called(path, manifestfile)

	projectName, err := p.ProjectName(path)
	if err != nil {
		return err
	}

	m, err := p.ManifestLoad(path, manifestfile)
	if err != nil {
		return err
	}

	nameTags, err := p.ManifestBuild(path, manifestfile)
	if err != nil {
		return err
	}

	// introspect environment for containers that are linked to
	nameEnv := map[string]map[string]string{}
	for serviceName, entry := range *m {
		for _, l := range entry.Links {
			tag, ok := nameTags[l]
			if !ok {
				return fmt.Errorf("%q link to %q is invalid", serviceName, l)
			}

			// inspect image for Dockerfile ENV LINK_ vars
			env, err := p.ImageInspect(tag)
			if err != nil {
				return err
			}

			nameEnv[l] = env

			// inspect manifest entry for overridden or filled in LINK_ vars
			env = map[string]string{}
			switch e := (*m)[l].Environment.(type) {
			case []string:
				for _, kv := range e {
					parts := strings.SplitN(kv, "=", 2)
					if len(parts) == 2 {
						env[parts[0]] = parts[1]
					}
				}
			}

			for k, v := range env {
				if strings.HasPrefix(k, "LINK_") {
					nameEnv[l][k] = v
				}
			}
		}
	}

	// run containers
	for serviceName, entry := range *m {
		name := fmt.Sprintf("%s-%s", projectName, serviceName)
		tag := fmt.Sprintf("%s/%s", projectName, serviceName)

		cmd := []string{}
		switch c := entry.Command.(type) {
		case string:
			if c != "" {
				cmd = append(cmd, "sh", "-c", c)
			}
		case []string:
			cmd = append(cmd, c...)
		}

		ports := []string{}
		switch p := entry.Ports.(type) {
		case string:
			if p != "" {
				ports = append(ports, p)
			}
		case []string:
			ports = append(ports, p...)
		}

		env := map[string]string{}
		switch e := entry.Environment.(type) {
		case []string:
			for _, kv := range e {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) == 2 {
					env[parts[0]] = parts[1]
				}
			}
		}

		for _, l := range entry.Links {
			for k, v := range nameEnv[l] {
				rk := strings.Replace(k, "LINK_", fmt.Sprintf("%s_", strings.ToUpper(l)), 1)
				env[rk] = v
			}
		}

		err := p.ProcessRun(tag, cmd, name, ports, env)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *TestProviderRunner) ProcessRun(tag string, args []string, name string, ports []string, env map[string]string) error {
	p.Called(tag, args, name, ports, env)
	return nil
}

func (p *TestProviderRunner) ProjectName(path string) (string, error) {
	args := p.Called(path)
	return args.String(0), args.Error(1)
}
