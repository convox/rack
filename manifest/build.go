package manifest

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (m *Manifest) Build(dir, appName string, s Stream, cache bool) error {
	pulls := map[string][]string{}
	builds := []Service{}

	for _, service := range m.runOrder() {
		dockerFile := service.Build.Dockerfile
		if dockerFile == "" {
			dockerFile = service.Dockerfile
		}
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

		if !cache {
			args = append(args, "--no-cache")
		}

		context := filepath.Join(dir, coalesce(service.Build.Context, "."))
		dockerFile := coalesce(service.Dockerfile, "Dockerfile")
		dockerFile = coalesce(service.Build.Dockerfile, dockerFile)

		args = append(args, "-f", fmt.Sprintf("%s/%s", context, dockerFile))
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

		if !cache || len(output) == 0 {
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
