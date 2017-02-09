package manifest

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type BuildOptions struct {
	Cache       bool
	CacheDir    string
	Environment map[string]string
	Service     string
	Verbose     bool
}

func (m *Manifest) Build(dir, appName string, s Stream, opts BuildOptions) error {
	pulls := map[string][]string{}
	builds := []Service{}

	services, err := m.runOrder(opts.Service)
	if err != nil {
		return err
	}

	for _, service := range services {
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
			if err := DefaultRunner.Run(s, Docker("tag", bc, service.Tag(appName)), RunnerOptions{Verbose: opts.Verbose}); err != nil {
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
		dockerFile = filepath.Join(context, dockerFile)

		if opts.CacheDir != "" {
			hash := service.Build.Hash()

			lcd := filepath.Join(dir, ".cache", "build")

			if err := os.MkdirAll(lcd, 0755); err != nil {
				s <- fmt.Sprintf("cache error: %s", err)
			}

			exec.Command("rm", "-rf", lcd).Run()
			exec.Command("cp", "-a", filepath.Join(opts.CacheDir, hash), lcd).Run()
			exec.Command("mkdir", "-p", lcd).Run()
		}

		bargs := map[string]string{}

		for k, v := range service.Build.Args {
			bargs[k] = v
		}

		dba, err := buildArgs(dockerFile)
		if err != nil {
			return err
		}

		for _, ba := range dba {
			if v, ok := opts.Environment[ba]; ok {
				bargs[ba] = v
			}
		}

		bargNames := []string{}

		for k := range bargs {
			bargNames = append(bargNames, k)
		}

		sort.Strings(bargNames)

		for _, name := range bargNames {
			args = append(args, "--build-arg", fmt.Sprintf("%s=%s", name, bargs[name]))
		}

		args = append(args, "-f", dockerFile)
		args = append(args, "-t", service.Tag(appName))
		args = append(args, context)

		if err := DefaultRunner.Run(s, Docker(args...), RunnerOptions{Verbose: opts.Verbose}); err != nil {
			return fmt.Errorf("build error: %s", err)
		}

		if opts.CacheDir != "" {
			hash := service.Build.Hash()

			if err := DefaultRunner.Run(s, Docker("create", "--name", hash, service.Tag(appName))); err != nil {
				s <- fmt.Sprintf("cache error: %s", err)
			}

			exec.Command("rm", "-rf", filepath.Join(opts.CacheDir, hash)).Run()

			if err := DefaultRunner.Run(s, Docker("cp", fmt.Sprintf("%s:/var/cache/build", hash), filepath.Join(opts.CacheDir, hash))); err != nil {
				s <- fmt.Sprintf("cache error: %s", err)
			}

			if err := DefaultRunner.Run(s, Docker("rm", hash)); err != nil {
				s <- fmt.Sprintf("cache error: %s", err)
			}
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
			if err := DefaultRunner.Run(s, Docker("pull", image), RunnerOptions{Verbose: opts.Verbose}); err != nil {
				return fmt.Errorf("build error: %s", err)
			}
		}
		for _, tag := range tags {
			if err := DefaultRunner.Run(s, Docker("tag", image, tag), RunnerOptions{Verbose: opts.Verbose}); err != nil {
				return fmt.Errorf("build error: %s", err)
			}
		}
	}

	return nil
}

func buildArgs(dockerfile string) ([]string, error) {
	args := []string{}

	data, err := ioutil.ReadFile(dockerfile)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())

		if len(parts) < 1 {
			continue
		}

		switch parts[0] {
		case "ARG":
			args = append(args, strings.SplitN(parts[1], "=", 2)[0])
		}
	}

	return args, nil
}
