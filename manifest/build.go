package manifest

import (
	"fmt"
	"time"
)

func (m *Manifest) Build(dir, appName string, s Stream, cache bool) error {
	pulls := map[string]string{}
	builds := []Service{}

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

		if !cache {
			args = append(args, "--no-cache")
		}

		context := coalesce(service.Build.Context, ".")
		dockerFile := coalesce(service.Dockerfile, "Dockerfile")
		dockerFile = coalesce(service.Build.Dockerfile, dockerFile)

		args = append(args, "-f", fmt.Sprintf("%s/%s", context, dockerFile))
		args = append(args, "-t", service.Tag(appName))
		args = append(args, context)

		if err := DefaultRunner.Run(s, Docker(args...)); err != nil {
			return fmt.Errorf("build error: %s", err)
		}
	}

	for image, tag := range pulls {
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

		if err := DefaultRunner.Run(s, Docker("tag", image, tag)); err != nil {
			return fmt.Errorf("build error: %s", err)
		}
	}

	return nil
}

func pushSync(s Stream, local, remote string) error {
	err := run(s, Docker("tag", local, remote))
	if err != nil {
		return err
	}

	return run(s, Docker("push", remote))
}

const (
	pushRetryLimit = 5
	pushRetryDelay = 30
)

// Push will push the image for a given process up to the appropriate registry
func (m *Manifest) Push(stream Stream, app, registry, tag string, flatten string) error {
	if tag == "" {
		tag = "latest"
	}

	for _, s := range m.runOrder() {
		local := fmt.Sprintf("%s/%s", app, s.Name)
		remote := fmt.Sprintf("%s/%s-%s:%s", registry, app, s.Name, tag)

		if flatten != "" {
			remote = fmt.Sprintf("%s/%s:%s", registry, flatten, fmt.Sprintf("%s.%s", s.Name, tag))
		}

		for i := 1; i <= pushRetryLimit; i++ {
			if err := DefaultRunner.Run(stream, Docker("tag", local, remote)); err != nil {
				return fmt.Errorf("could not tag build: %s", err)
			}

			if err := DefaultRunner.Run(stream, Docker("push", remote)); err == nil {
				break
			}

			fmt.Printf("An error occurred while trying to push %s/%s\n", app, s.Name)
			fmt.Printf("Retrying in %d seconds (attempt %d/%d)\n", pushRetryDelay, i, pushRetryLimit)
			time.Sleep(pushRetryDelay * time.Second)
		}
	}

	return nil
}
