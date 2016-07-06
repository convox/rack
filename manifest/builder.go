package manifest

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func buildSync(source, tag string, cache bool, dockerfile string) error {
	args := []string{"build", "-t", tag}

	// if called with `convox build --no-cache`, assume intent to build from scratch.
	// So both pull latest images from DockerHub and build without cache
	if !cache {
		args = append(args, "--pull")
		args = append(args, "--no-cache")
	}

	if dockerfile != "" {
		args = append(args, "-f", filepath.Join(source, dockerfile))
	}

	args = append(args, source)

	return runBuilder("docker", args...)
}

func pullSync(image string) error {
	return runBuilder("docker", "pull", image)
}

func pushSync(local, remote string) error {
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

func (m *Manifest) BuildRack(app, dir string, cache bool) []error {
	builds := map[string]string{}
	pulls := []string{}
	tags := map[string]string{}

	for name, entry := range m.Services {
		tag := fmt.Sprintf("%s/%s", app, name)

		switch {
		case entry.Build.Context != "":
			abs, err := filepath.Abs(filepath.Join(entry.Build.Context, entry.Build.Dockerfile))

			if err != nil {
				return []error{err}
			}

			sym, err := filepath.EvalSymlinks(abs)
			if err != nil {
				return []error{err}
			}

			df := "Dockerfile"
			if entry.Dockerfile != "" {
				df = entry.Dockerfile
			}

			sym = filepath.Join(sym, df)

			if _, ok := builds[sym]; !ok {
				builds[sym] = randomString("convox-", 10)
			}

			tags[tag] = builds[sym]
		case entry.Image != "":
			err := exec.Command("docker", "inspect", entry.Image).Run()

			if err != nil || !cache {
				pulls = append(pulls, entry.Image)
			}

			tags[tag] = entry.Image
		}
	}

	errors := []error{}

	for path, tag := range builds {
		source, dockerfile := filepath.Split(path)
		err := buildSync(source, tag, cache, dockerfile)

		if err != nil {
			return []error{err}
		}
	}

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	for _, image := range pulls {
		var pullErr error
		var backOff = 1

		for i := 0; i < 5; i++ {
			if i != 0 {
				log.Printf("A pull error occurred for: %s\n", image)
				log.Printf("Retrying in %d seconds...\n", backOff)
				time.Sleep(time.Duration(backOff) * time.Second)
				backOff = ((backOff + r1.Intn(10)) * (i))
			}
			pullErr = pullSync(image)
			if pullErr == nil {
				break
			}
		}

		if pullErr != nil {
			return []error{pullErr}
		}
	}

	if len(errors) > 0 {
		return errors
	}

	// tag in alphabetical order for testability
	mk := make([]string, len(tags))
	i := 0
	for k, _ := range tags {
		mk[i] = k
		i++
	}
	sort.Strings(mk)

	for _, to := range mk {
		from := tags[to]
		// for to, from := range tags {
		err := runBuilder("docker", "tag", from, to)

		if err != nil {
			return []error{err}
		}
	}

	return []error{}
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
