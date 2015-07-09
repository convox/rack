package manifest

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type Manifest map[string]ManifestEntry

type ManifestEntry struct {
	Build       string      `yaml:"build,omitempty"`
	Image       string      `yaml:"image,omitempty"`
	Command     interface{} `yaml:"command,omitempty"`
	Environment []string    `yaml:"environment,omitempty"`
	EnvFile     string      `yaml:"env_file,omitempty"`
	Links       []string    `yaml:"links,omitempty"`
	Ports       []string    `yaml:"ports,omitempty"`
	Volumes     []string    `yaml:"volumes,omitempty"`
}

func Generate(dir string) (*Manifest, error) {
	err := os.Chdir(dir)

	if err != nil {
		return nil, err
	}

	var m *Manifest

	switch {
	case exists(filepath.Join(dir, "docker-compose.yml")):
		m, err = buildDockerCompose(dir)
	case exists(filepath.Join(dir, "Dockerfile")):
		m, err = buildDockerfile(dir)
	case exists(filepath.Join(dir, "Procfile")):
		m, err = buildProcfile(dir)
	default:
		return nil, fmt.Errorf("could not find any manifests")
	}

	if err != nil {
		return nil, err
	}

	return m, nil
}

func buildAsync(source, tag string, ch chan error) {
	ch <- run("docker", "build", "-t", tag, source)
}

func (m *Manifest) Build(app string) []error {
	ch := make(chan error)

	builds := map[string]string{}
	tags := map[string]string{}

	for name, entry := range *m {
		tag := fmt.Sprintf("%s/%s", app, name)

		switch {
		case entry.Build != "":
			if _, ok := builds[entry.Build]; !ok {
				builds[entry.Build] = randomString(10)
			}

			tags[tag] = builds[entry.Build]
		case entry.Image != "":
			tags[tag] = entry.Image
		}
	}

	errors := []error{}

	for source, tag := range builds {
		go buildAsync(source, tag, ch)
	}

	for i := 0; i < len(builds); i++ {
		if err := <-ch; err != nil {
			errors = append(errors, err)
		}
	}

	for to, from := range tags {
		run("docker", "tag", "-f", from, to)
	}

	return errors
}

func (m *Manifest) Run(app string) []error {
	ch := make(chan error)

	for _, name := range m.runOrder() {
		go (*m)[name].RunAsync(m.prefixForEntry(name), app, name, ch)
		time.Sleep(200 * time.Millisecond)
	}

	errors := []error{}

	for i := 0; i < len(*m); i++ {
		if err := <-ch; err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

func (m *Manifest) Write(filename string) error {
	data, err := yaml.Marshal(m)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

func (m *Manifest) prefixForEntry(name string) string {
	longest := 0

	for name, _ := range *m {
		if len(name) > longest {
			longest = len(name)
		}
	}

	return name + strings.Repeat(" ", longest-len(name))
}

func (m *Manifest) runOrder() []string {
	rs := RunSorter{manifest: *m, names: make([]string, len(*m))}

	i := 0

	for name, _ := range *m {
		rs.names[i] = name
		i++
	}

	sort.Sort(rs)

	return rs.names
}

func (me ManifestEntry) RunAsync(prefix, app, process string, ch chan error) {
	tag := fmt.Sprintf("%s/%s", app, process)
	name := fmt.Sprintf("%s-%s", app, process)

	query("docker", "rm", "-f", name)

	args := []string{"run", "-i", "--name", name, "--rm=true"}

	for _, env := range me.Environment {
		if strings.Index(env, "=") > -1 {
			args = append(args, "-e", env)
		} else {
			args = append(args, "-e", fmt.Sprintf("%s=%s", env, os.Getenv(env)))
		}
	}

	for _, link := range me.Links {
		args = append(args, "--link", fmt.Sprintf("%s-%s:%s", app, link, link))
	}

	for _, port := range me.Ports {
		args = append(args, "-p", port)
	}

	for _, volume := range me.Volumes {
		args = append(args, "-v", volume)
	}

	args = append(args, tag)

	switch cmd := me.Command.(type) {
	case string:
		if cmd != "" {
			args = append(args, "sh", "-c", cmd)
		}
	case []string:
		args = append(args, cmd...)
	}

	ch <- runPrefix(prefix, "docker", args...)
}

func injectDockerfile(dir string) error {
	detect := ""

	switch {
	case exists(filepath.Join(dir, "Gemfile.lock")):
		detect = "ruby"
	default:
		detect = "unknown"
	}

	data, err := Asset(fmt.Sprintf("data/Dockerfile.%s", detect))

	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(dir, "Dockerfile"), data, 0644)
}

func exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

func query(executable string, args ...string) ([]byte, error) {
	return exec.Command(executable, args...).CombinedOutput()
}

func run(executable string, args ...string) error {
	cmd := exec.Command(executable, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runPrefix(prefix, executable string, args ...string) error {
	cmd := exec.Command(executable, args...)

	stdout, err := cmd.StdoutPipe()

	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()

	if err != nil {
		return err
	}

	err = cmd.Start()

	if err != nil {
		return err
	}

	ch := make(chan error)

	go prefixReader(prefix, stdout, ch)
	go prefixReader(prefix, stderr, ch)

	if err := <-ch; err != nil {
		return err
	}

	if err := <-ch; err != nil {
		return err
	}

	return cmd.Wait()
}

func prefixReader(prefix string, r io.Reader, ch chan error) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		fmt.Printf("%s | %s\n", prefix, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		ch <- err
	}

	ch <- nil
}

var randomAlphabet = []rune("abcdefghijklmnopqrstuvwxyz")

func randomString(size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = randomAlphabet[rand.Intn(len(randomAlphabet))]
	}
	return string(b)
}
