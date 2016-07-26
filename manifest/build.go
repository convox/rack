package manifest

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"
)

func (m *Manifest) Build(dir, appName string, s Stream, noCache bool) error {
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

		if noCache {
			args = append(args, "--no-cache")
		}

		context := coalesce(service.Build.Context, ".")
		dockerFile := coalesce(service.Build.Dockerfile, "Dockerfile")

		args = append(args, "-f", fmt.Sprintf("%s/%s", context, dockerFile))
		args = append(args, "-t", service.Tag(appName))
		args = append(args, context)
		run(s, Docker(args...))
	}

	for image, tag := range pulls {
		args := []string{"pull"}

		args = append(args, image)

		run(s, Docker(args...))
		run(s, Docker("tag", image, tag))
	}

	return nil
}

func pullSync(image string) error {
	return runBuilder("docker", "pull", image)
}

func pushSync(local, remote string) error {
	log.Print("PUSH SYNC")
	log.Print(local)
	log.Print(remote)
	err := runBuilder("docker", "tag", local, remote)

	if err != nil {
		return err
	}

	err = runBuilder("docker", "push", remote)

	if err != nil {
		return err
	}

	return nil
}

func runBuilder(executable string, args ...string) error {
	os.Stdout.Write([]byte(fmt.Sprintf("RUNNING: %s %s\n", executable, strings.Join(args, " "))))

	cmd := exec.Command(executable, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

var randomAlphabet = []rune("abcdefghijklmnopqrstuvwxyz")

func randomString(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = randomAlphabet[rand.Intn(len(randomAlphabet))]
	}
	return prefix + string(b)
}

func (m *Manifest) Push(app, registry, tag string, flatten string) []error {
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
			pushErr = pushSync(local, remote)
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
