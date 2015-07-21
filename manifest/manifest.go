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

	"github.com/convox/cli/Godeps/_workspace/src/github.com/fatih/color"
	yaml "github.com/convox/cli/Godeps/_workspace/src/gopkg.in/yaml.v2"
)

var (
	Stdout       = io.Writer(os.Stdout)
	Stderr       = io.Writer(os.Stderr)
	Execer       = exec.Command
	SignalWaiter = waitForSignal
)

var Colors = []color.Attribute{color.FgCyan, color.FgYellow, color.FgGreen, color.FgMagenta, color.FgBlue}

type Manifest map[string]ManifestEntry

type ManifestEntry struct {
	Build       string      `yaml:"build,omitempty"`
	Image       string      `yaml:"image,omitempty"`
	Command     interface{} `yaml:"command,omitempty"`
	Environment []string    `yaml:"environment,omitempty"`
	Links       []string    `yaml:"links,omitempty"`
	Ports       interface{} `yaml:"ports,omitempty"`
	Volumes     []string    `yaml:"volumes,omitempty"`
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func Generate(dir string) (*Manifest, error) {
	wd, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	defer os.Chdir(wd)

	err = os.Chdir(dir)

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
		m, err = buildDefault(dir)
	}

	if err != nil {
		return nil, err
	}

	return m, nil
}

func buildAsync(source, tag string, ch chan error) {
	ch <- buildSync(source, tag)
}

func buildSync(source, tag string) error {
	return run("docker", "build", "-t", tag, source)
}

func pullAsync(image string, ch chan error) {
	ch <- pullSync(image)
}

func pullSync(image string) error {
	return run("docker", "pull", image)
}

func pushAsync(local, remote string, ch chan error) {
	ch <- pushSync(local, remote)
}

func pushSync(local, remote string) error {
	err := run("docker", "tag", "-f", local, remote)

	if err != nil {
		return err
	}

	err = run("docker", "push", remote)

	if err != nil {
		return err
	}

	return nil
}

func (m *Manifest) Build(app, dir string) []error {
	builds := map[string]string{}
	pulls := []string{}
	tags := map[string]string{}

	for name, entry := range *m {
		tag := fmt.Sprintf("%s/%s", app, name)

		switch {
		case entry.Build != "":
			abs, err := filepath.Abs(filepath.Join(dir, entry.Build))

			if err != nil {
				return []error{err}
			}

			sym, err := filepath.EvalSymlinks(abs)

			if err != nil {
				return []error{err}
			}
			if _, ok := builds[sym]; !ok {
				builds[sym] = randomString(10)
			}

			tags[tag] = builds[sym]
		case entry.Image != "":
			pulls = append(pulls, entry.Image)
			tags[tag] = entry.Image
		}
	}

	errors := []error{}

	for source, tag := range builds {
		err := buildSync(source, tag)

		if err != nil {
			return []error{err}
		}
	}

	for _, image := range pulls {
		err := pullSync(image)

		if err != nil {
			return []error{err}
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
		err := run("docker", "tag", "-f", from, to)

		if err != nil {
			return []error{err}
		}
	}

	return []error{}
}

func (m *Manifest) MissingEnvironment() []string {
	existing := map[string]bool{}
	missingh := map[string]bool{}

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)

		if len(parts) == 2 {
			existing[parts[0]] = true
		}
	}

	for _, entry := range *m {
		for _, env := range entry.Environment {
			if strings.Index(env, "=") == -1 {
				if !existing[env] {
					missingh[env] = true
				}
			}
		}
	}

	missing := []string{}

	for mm, _ := range missingh {
		missing = append(missing, mm)
	}

	sort.Strings(missing)

	return missing
}

func (m *Manifest) PortsWanted() ([]string, error) {
	ports := make([]string, 0)

	for _, entry := range *m {
		if pp, ok := entry.Ports.([]interface{}); ok {
			for _, port := range pp {
				if p, ok := port.(string); ok {
					parts := strings.SplitN(p, ":", 2)

					if len(parts) == 2 {
						ports = append(ports, parts[0])
					}
				}
			}
		}
	}

	return ports, nil
}

func (m *Manifest) Push(app, registry, auth, tag string) []error {
	// ch := make(chan error)

	if auth != "" {
		err := run("docker", "login", "-e", "user@convox.io", "-u", "convox", "-p", auth, registry)

		if err != nil {
			return []error{err}
		}
	}

	if tag == "" {
		tag = "latest"
	}

	for name, _ := range *m {
		local := fmt.Sprintf("%s/%s", app, name)
		remote := fmt.Sprintf("%s/%s-%s:%s", registry, app, name, tag)

		err := pushSync(local, remote)

		if err != nil {
			return []error{err}
		}
	}

	return []error{}
}

func (m *Manifest) Raw() ([]byte, error) {
	return yaml.Marshal(m)
}

func (m *Manifest) Run(app string) []error {
	ch := make(chan error)

	missing := m.MissingEnvironment()

	if len(missing) > 0 {
		return []error{fmt.Errorf("env expected: %s", strings.Join(missing, ", "))}
	}

	for _, entry := range *m {
		for i, env := range entry.Environment {
			if strings.Index(env, "=") == -1 {
				entry.Environment[i] = fmt.Sprintf("%s=%s", env, os.Getenv(env))
			}
		}
	}

	// Set up channel on which to send signal notifications.
	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt, os.Kill)

	for i, name := range m.runOrder() {
		go (*m)[name].runAsync(m.prefixForEntry(name, i), app, name, ch)
		time.Sleep(1000 * time.Millisecond)
	}

	errors := []error{}

	for i := 0; i < len(*m); i++ {
		if err := <-ch; err != nil {
			errors = append(errors, err)
		}
	}

	// err := SignalWaiter(c)
	// errors = append(errors, err)

	return errors
}

func waitForSignal(c chan os.Signal) error {
	s := <-c
	return fmt.Errorf("signal %s", s)
}

func (m *Manifest) Write(filename string) error {
	data, err := m.Raw()

	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

func (m *Manifest) prefixForEntry(name string, pos int) string {
	longest := 0

	for name, _ := range *m {
		if len(name) > longest {
			longest = len(name)
		}
	}

	c := color.New(Colors[pos%len(Colors)]).SprintFunc()

	return c(name + strings.Repeat(" ", longest-len(name)) + " |")
}

func (m *Manifest) runOrder() []string {
	unsorted := []string{}

	for name, _ := range *m {
		unsorted = append(unsorted, name)
	}

	sort.Strings(unsorted)

	sorted := []string{}

	for len(sorted) < len(unsorted) {
		for _, name := range unsorted {
			found := false

			for _, n := range sorted {
				if n == name {
					found = true
					break
				}
			}

			if found {
				continue
			}

			resolved := true

			for _, link := range (*m)[name].Links {
				lname := strings.Split(link, ":")[0]

				found := false

				for _, n := range sorted {
					if n == lname {
						found = true
					}
				}

				if !found {
					resolved = false
					break
				}
			}

			if resolved {
				sorted = append(sorted, name)
			}
		}
	}

	return sorted
}

func (me ManifestEntry) runAsync(prefix, app, process string, ch chan error) {
	tag := fmt.Sprintf("%s/%s", app, process)
	name := fmt.Sprintf("%s-%s", app, process)

	query("docker", "rm", "-f", name)

	args := []string{"run", "-i", "--name", name}

	for _, env := range me.Environment {
		if strings.Index(env, "=") > -1 {
			args = append(args, "-e", env)
		} else {
			args = append(args, "-e", fmt.Sprintf("%s=%s", env, os.Getenv(env)))
		}
	}

	for _, link := range me.Links {
		parts := strings.Split(link, ":")

		switch len(parts) {
		case 1:
			args = append(args, "--link", fmt.Sprintf("%s-%s:%s", app, link, link))
		case 2:
			args = append(args, "--link", fmt.Sprintf("%s-%s:%s", app, parts[0], parts[1]))
		default:
		}
	}

	switch t := me.Ports.(type) {
	case []string:
		for _, port := range t {
			args = append(args, "-p", port)
		}
	case []interface{}:
		for _, port := range t {
			if p, ok := port.(string); ok {
				args = append(args, "-p", p)
			}
		}
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

func exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

func injectDockerfile(dir string) error {
	detect := ""

	switch {
	case exists(filepath.Join(dir, "package.json")):
		detect = "node"
	case exists(filepath.Join(dir, "config/application.rb")):
		detect = "rails"
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

func query(executable string, args ...string) ([]byte, error) {
	return Execer(executable, args...).CombinedOutput()
}

func outputWithPrefix(prefix string, r io.Reader, ch chan error) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		fmt.Printf("%s %s\n", prefix, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		ch <- err
	}

	ch <- nil
}

func run(executable string, args ...string) error {
	Stdout.Write([]byte(fmt.Sprintf("RUNNING: %s %s\n", executable, strings.Join(args, " "))))

	cmd := Execer(executable, args...)
	cmd.Stdout = Stdout
	cmd.Stderr = Stderr
	return cmd.Run()
}

func runPrefix(prefix, executable string, args ...string) error {
	fmt.Printf("%s running: %s %s\n", prefix, executable, strings.Join(args, " "))

	cmd := Execer(executable, args...)

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

	go outputWithPrefix(prefix, stdout, ch)
	go outputWithPrefix(prefix, stderr, ch)

	if err := <-ch; err != nil {
		return err
	}

	if err := <-ch; err != nil {
		return err
	}

	return cmd.Wait()
}

var randomAlphabet = []rune("abcdefghijklmnopqrstuvwxyz")

func randomString(size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = randomAlphabet[rand.Intn(len(randomAlphabet))]
	}
	return string(b)
}
