package manifest

import (
	"fmt"
	"path/filepath"
	"strings"
)

type BuildOptions struct {
	Cache   bool
	Service string
}

func (m *Manifest) Build(dir, appName string, s Stream, opts BuildOptions) error {
	pulls := map[string][]string{}
	builds := []Service{}

	services, err := m.runOrder(opts.Service)
	if err != nil {
		return err
	}

	for _, service := range services {
		if image := service.Image; image != "" {
			// make the implicit :latest explicit for caching/pulling
			sp := strings.Split(image, "/")
			if !strings.Contains(sp[len(sp)-1], ":") {
				image = image + ":latest"
			}
			pulls[image] = append(pulls[image], service.Tag(appName))
		} else {
			builds = append(builds, service)
		}
	}

	buildCache := map[string]string{}

	for _, service := range builds {
		if bc, ok := buildCache[service.Build.Hash()]; ok {
			if err := DefaultRunner.Run(s, Docker("tag", bc, service.Tag(appName))); err != nil {
				return fmt.Errorf("build error: %s", err)
			}
			continue
		}

		args := []string{"build"}

		if !opts.Cache {
			args = append(args, "--no-cache")
		}

		context := filepath.Join(dir, coalesce(service.Build.Context, "."))
		dockerFile := coalesce(service.Dockerfile, "Dockerfile")
		dockerFile = coalesce(service.Build.Dockerfile, dockerFile)

		args = append(args, "-f", filepath.Join(context, dockerFile))
		args = append(args, "-t", service.Tag(appName))
		args = append(args, context)

		if err := DefaultRunner.Run(s, Docker(args...)); err != nil {
			return fmt.Errorf("build error: %s", err)
		}

		buildCache[service.Build.Hash()] = service.Tag(appName)
	}

	for image, tags := range pulls {
		args := []string{"pull"}

		output, err := DefaultRunner.CombinedOutput(Docker("images", "-q", image))
		if err != nil {
			return err
		}

		args = append(args, image)

		if !opts.Cache || len(output) == 0 {
			if err := DefaultRunner.Run(s, Docker("pull", image)); err != nil {
				return fmt.Errorf("build error: %s", err)
			}
		}
		for _, tag := range tags {
			if err := DefaultRunner.Run(s, Docker("tag", image, tag)); err != nil {
				return fmt.Errorf("build error: %s", err)
			}
		}
	}

	return nil
}
