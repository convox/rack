package manifest

import (
	"fmt"
	"log"
	"math/rand"
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
		DefaultRunner.Run(s, Docker(args...))
	}

	for image, tag := range pulls {
		args := []string{"pull"}
		args = append(args, image)
		DefaultRunner.Run(s, Docker(args...))
		DefaultRunner.Run(s, Docker("tag", image, tag))
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

func (m *Manifest) Push(s Stream, app, registry, tag string, flatten string) []error {
	if tag == "" {
		tag = "latest"
	}

	for name, _ := range m.Services {
		local := fmt.Sprintf("%s/%s", app, name)
		remote := fmt.Sprintf("%s/%s-%s:%s", registry, app, name, tag)

		if flatten != "" {
			remote = fmt.Sprintf("%s/%s:%s", registry, flatten, fmt.Sprintf("%s.%s", name, tag))
		}

		var pushErr error
		var backOff = 1
		s1 := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(s1)

		for i := 0; i < 5; i++ {
			if i != 0 {
				log.Printf("A push error occurred for %s/%s\n", app, name)
				log.Printf("Retrying in %d seconds...\n", backOff)
				time.Sleep(time.Duration(backOff) * time.Second)
				backOff = ((backOff + r1.Intn(10)) * (i))
			}
			pushErr = pushSync(s, local, remote)
			if pushErr == nil {
				break
			}
		}

		if pushErr != nil {
			return []error{pushErr}
		}
	}

	return []error{}
}
