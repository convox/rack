package manifest

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/Godeps/_workspace/src/github.com/fatih/color"
	yaml "github.com/convox/rack/Godeps/_workspace/src/gopkg.in/yaml.v2"
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
	Entrypoint  string      `yaml:"entrypoint,omitempty"`
	Environment interface{} `yaml:"environment,omitempty"`
	Links       []string    `yaml:"links,omitempty"`
	Ports       interface{} `yaml:"ports,omitempty"`
	Volumes     []string    `yaml:"volumes,omitempty"`
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func Init(dir string) (changed []string, err error) {
	wd, err := os.Getwd()

	if err != nil {
		return nil, err
	}

	defer os.Chdir(wd)

	err = os.Chdir(dir)

	if err != nil {
		return nil, err
	}

	switch {
	case exists(filepath.Join(dir, "docker-compose.yml")):
		fmt.Println("Manifest already exists")
	case exists(filepath.Join(dir, "Dockerfile")):
		changed, err = initDockerfile(dir)
	case exists(filepath.Join(dir, "Procfile")):
		changed, err = initProcfile(dir)
	default:
		changed, err = initDefault(dir)
	}

	if err != nil {
		return nil, err
	}

	return changed, nil
}

func Read(dir string) (*Manifest, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, "docker-compose.yml"))

	if err != nil {
		return nil, fmt.Errorf("file not found: docker-compose.yml")
	}

	var m Manifest

	err = yaml.Unmarshal(data, &m)

	if err != nil {
		return nil, err
	}

	if denv := filepath.Join(dir, ".env"); exists(denv) {
		data, err := ioutil.ReadFile(denv)

		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(bytes.NewReader(data))

		for scanner.Scan() {
			if strings.Index(scanner.Text(), "=") > -1 {
				parts := strings.SplitN(scanner.Text(), "=", 2)

				err := os.Setenv(parts[0], parts[1])

				if err != nil {
					return nil, err
				}
			}
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	for name, entry := range m {
		for i, volume := range entry.Volumes {
			parts := strings.Split(volume, ":")

			for j, part := range parts {
				if !filepath.IsAbs(part) {
					parts[j] = filepath.Join(dir, part)
				}
			}

			m[name].Volumes[i] = strings.Join(parts, ":")
		}
	}

	return &m, nil
}

func buildSync(source, tag string, cache bool) error {
	args := []string{"build", "-t", tag}

	// if called with `convox build --no-cache`, assume intent to build from scratch.
	// So both pull latest images from DockerHub and build without cache
	if !cache {
		args = append(args, "--pull")
		args = append(args, "--no-cache")
	}

	args = append(args, source)

	return run("docker", args...)
}

func pullSync(image string) error {
	return run("docker", "pull", image)
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

func (m *Manifest) Build(app, dir string, cache bool) []error {
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
		err := buildSync(source, tag, cache)

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

func (me *ManifestEntry) ResolvedEnvironment() []string {
	r := []string{}

	for _, env := range me.EnvironmentArray() {
		if strings.Index(env, "=") == -1 {
			env = fmt.Sprintf("%s=%s", env, os.Getenv(env))
		}
		r = append(r, env)
	}

	return r
}

func (me *ManifestEntry) EnvironmentArray() []string {
	var arr []string
	switch t := me.Environment.(type) {
	case map[interface{}]interface{}:
		for k, v := range t {
			arr = append(arr, fmt.Sprintf("%s=%s", k, v))
		}
	case []interface{}:
		for _, s := range t {
			arr = append(arr, s.(string))
		}
	default:
		// Unknown type. No action.
	}
	return arr
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
		for _, env := range entry.EnvironmentArray() {
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
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt, os.Kill)

	go func() {
		for _ = range sigch {
			order := m.runOrder()

			sort.Sort(sort.Reverse(sort.StringSlice(order)))

			for _, name := range order {
				Execer("docker", "kill", containerName(app, name)).Run()
			}
			os.Exit(0)
		}
	}()

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

	c := color.New(Colors[pos%len(Colors)])

	c.EnableColor()

	return c.SprintFunc()(name + strings.Repeat(" ", longest-len(name)) + " |")
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

func containerName(app, process string) string {
	return fmt.Sprintf("%s-%s", app, process)
}

func (me ManifestEntry) runAsync(prefix, app, process string, ch chan error) {
	tag := fmt.Sprintf("%s/%s", app, process)
	name := containerName(app, process)

	query("docker", "rm", "-f", name)

	args := []string{"run", "-i", "--name", name}

	for _, env := range me.ResolvedEnvironment() {
		args = append(args, "-e", env)
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

	ports := []string{}

	switch t := me.Ports.(type) {
	case []string:
		for _, port := range t {
			ports = append(ports, port)
		}
	case []interface{}:
		for _, port := range t {
			ports = append(ports, fmt.Sprintf("%v", port))
		}
	}

	for _, port := range ports {
		switch len(strings.Split(port, ":")) {
		case 1:
			args = append(args, "-p", fmt.Sprintf("%s:%s", port, port))
		case 2:
			args = append(args, "-p", port)
		default:
			ch <- fmt.Errorf("unknown port declaration: %s", port)
			return
		}
	}

	for _, volume := range me.Volumes {
		args = append(args, "-v", volume)
	}

	if me.Entrypoint != "" {
		args = append(args, "--entrypoint", me.Entrypoint)
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
	case exists(filepath.Join(dir, ".meteor")):
		detect = "meteor"
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

var exposeEntryRegexp = regexp.MustCompile(`^EXPOSE\s+(\d+)`)

func initDockerfile(dir string) ([]string, error) {
	entry := ManifestEntry{
		Build: ".",
		Ports: []string{},
	}

	data, err := ioutil.ReadFile(filepath.Join(dir, "Dockerfile"))

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	current := 5000

	for scanner.Scan() {
		parts := exposeEntryRegexp.FindStringSubmatch(scanner.Text())

		if len(parts) > 1 {
			entry.Ports = append(entry.Ports.([]string), fmt.Sprintf("%d:%s", current, strings.Split(parts[1], "/")[0]))
			current += 100
		}
	}

	manifest := &Manifest{"main": entry}

	err = manifest.Write(filepath.Join(dir, "docker-compose.yml"))

	if err != nil {
		return nil, err
	}

	return []string{"docker-compose.yml"}, nil
}

var procfileEntryRegexp = regexp.MustCompile("^([A-Za-z0-9_]+):\\s*(.+)$")

func initProcfile(dir string) ([]string, error) {
	m := Manifest{}

	err := injectDockerfile(dir)

	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(filepath.Join(dir, "Procfile"))

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	current := 5000

	for scanner.Scan() {
		parts := procfileEntryRegexp.FindStringSubmatch(scanner.Text())

		if len(parts) > 0 {
			m[parts[1]] = ManifestEntry{
				Build:   ".",
				Command: parts[2],
				Ports:   []string{fmt.Sprintf("%d:3000", current)},
			}

			current += 100
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	err = m.Write(filepath.Join(dir, "docker-compose.yml"))

	if err != nil {
		return nil, err
	}

	return []string{"Dockerfile", "docker-compose.yml"}, nil
}

func initDefault(dir string) ([]string, error) {
	m := Manifest{}

	err := injectDockerfile(dir)

	if err != nil {
		return nil, err
	}

	m["main"] = ManifestEntry{
		Build: ".",
		Ports: []string{"5000:3000"},
	}

	err = m.Write(filepath.Join(dir, "docker-compose.yml"))

	if err != nil {
		return nil, err
	}

	return []string{"Dockerfile", "docker-compose.yml"}, nil
}
