package manifest

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/convox/rack/cmd/convox/stdcli"
)

func (m *Manifest) Build(dir string, s Stream, noCache bool) error {
	pulls := map[string]string{}
	builds := []Service{}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	appName := stdcli.ReadSetting("app")
	if appName == "" {
		appName = path.Base(abs)
	}

	appName = strings.ToLower(appName)

	for _, service := range m.Services {
		dockerFile := service.Build.Dockerfile
		if dockerFile == "" {
			dockerFile = service.Dockerfile
		}
		if service.Image != "" {
			pulls[service.Image] = service.Tag(appName)
		} else {
			builds = append(builds, service)
		}
	}

	for _, service := range builds {
		args := []string{"build"}

		if noCache {
			args = append(args, "--no-cache")
		}

		context := coalesce(service.Build.Context, ".")
		dockerFile := coalesce(service.Build.Dockerfile, "Dockerfile")

		args = append(args, "-f", fmt.Sprintf("%s/%s", context, dockerFile))
		args = append(args, "-t", service.Tag(appName))
		args = append(args, context)
		run(s, Docker(args...))
		// runPrefix(systemPrefix(m), Docker(args...))
	}

	for image, tag := range pulls {
		args := []string{"pull"}

		args = append(args, image)

		run(s, Docker(args...))
		run(s, Docker("tag", image, tag))
		// runPrefix(systemPrefix(m), Docker(args...))
		// runPrefix(systemPrefix(m), Docker("tag", image, tag))
	}

	return nil
}
