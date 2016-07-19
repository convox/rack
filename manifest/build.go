package manifest

import (
	"fmt"
	"strings"
)

func (m *Manifest) Build(dir string, s Stream, noCache bool) error {
	builds := map[string]string{}
	pulls := map[string]string{}

	for _, service := range m.Services {
		dockerFile := service.Build.Dockerfile
		if dockerFile == "" {
			dockerFile = service.Dockerfile
		}
		switch {
		case service.Build.Context != "":
			builds[fmt.Sprintf("%s|%s", service.Build.Context, coalesce(dockerFile, "Dockerfile"))] = service.Tag()
		case service.Image != "":
			pulls[service.Image] = service.Tag()
		}
	}

	for build, tag := range builds {
		parts := strings.SplitN(build, "|", 2)

		args := []string{"build"}

		if noCache {
			args = append(args, "--no-cache")
		}

		args = append(args, "-f", parts[1])
		args = append(args, "-t", tag)
		args = append(args, parts[0])
		builder := Docker(args...)
		builder.Dir = dir
		run(s, builder)
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
