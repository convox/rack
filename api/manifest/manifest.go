package manifest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/url"
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

//NOTE: these vars allow us to control other shell-outs during testing
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
	Dockerfile  string      `yaml:"dockerfile,omitempty"`
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

func Read(dir, filename string) (*Manifest, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, filename))

	if err != nil {
		return nil, fmt.Errorf("file not found: %s", filename)
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

			entry.Volumes[i] = strings.Join(parts, ":")
		}

		m[name] = entry
	}

	return &m, nil
}

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
	dockerfiles := map[string]string{}

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

			// Dockerfile can only be specified if Build is also specified
			if entry.Dockerfile != "" {
				dockerfiles[sym] = entry.Dockerfile
			}

		case entry.Image != "":
			pulls = append(pulls, entry.Image)
			tags[tag] = entry.Image
		}
	}

	errors := []error{}

	for source, tag := range builds {
		err := buildSync(source, tag, cache, dockerfiles[source])

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

func (me *ManifestEntry) ResolvedEnvironment(m *Manifest) ([]string, error) {
	r := []string{}

	linkedVars, err := me.ResolvedLinkVars(m)
	if err != nil {
		return r, err
	}

	for _, env := range me.EnvironmentArray() {
		// value is of form: `- KEY` without an explicit value so the
		// system looks it up
		if strings.Index(env, "=") == -1 {
			if val := linkedVars[env]; val != "" {
				delete(linkedVars, env)
				env = fmt.Sprintf("%s=%s", env, val)
			} else if val := os.Getenv(env); val != "" {
				env = fmt.Sprintf("%s=%s", env, val)
			}
		}
		r = append(r, env)
	}

	// appends the unused service links to the front of the array
	// so you still get them if you haven't declared them
	for key, value := range linkedVars {
		env := fmt.Sprintf("%s=%s", key, value)
		r = append([]string{env}, r...)
	}

	sort.Strings(r)

	return r, nil
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

// NOTE: this is the simpler approach:
//       build up the ENV from the declared links
//       assuming local dev is done on DOCKER_HOST
func (me *ManifestEntry) ResolvedLinkVars(m *Manifest) (map[string]string, error) {
	linkVars := make(map[string]string)

	if m == nil {
		return linkVars, nil
	}

	for _, link := range me.Links {
		linkEntry := (*m)[link]

		linkEntryEnv, err := getLinkEntryEnv(linkEntry)
		if err != nil {
			return linkVars, err
		}

		// get url parts from various places
		scheme := linkEntryEnv["LINK_SCHEME"]
		if scheme == "" {
			scheme = "tcp"
		}

		host, err := getDockerGateway()
		if err != nil {
			return linkVars, err
		}

		// we don't create a balancer without a port,
		// so we don't create a link url either
		port := resolveOtherPort(link, linkEntry)
		if port == "" {
			continue
		}

		linkUrl := url.URL{
			Scheme: scheme,
			Host:   host + ":" + port,
			Path:   linkEntryEnv["LINK_PATH"],
		}

		if linkEntryEnv["LINK_USERNAME"] != "" || linkEntryEnv["LINK_PASSWORD"] != "" {
			linkUrl.User = url.UserPassword(linkEntryEnv["LINK_USERNAME"], linkEntryEnv["LINK_PASSWORD"])
		}

		prefix := strings.ToUpper(link) + "_"
		linkVars[prefix+"URL"] = linkUrl.String()
		linkVars[prefix+"HOST"] = host
		linkVars[prefix+"SCHEME"] = scheme
		linkVars[prefix+"PORT"] = port
		linkVars[prefix+"USERNAME"] = linkEntryEnv["LINK_USERNAME"]
		linkVars[prefix+"PASSWORD"] = linkEntryEnv["LINK_PASSWORD"]
		linkVars[prefix+"PATH"] = linkEntryEnv["LINK_PATH"]
	}

	return linkVars, nil
}

func (m *Manifest) MissingEnvironment() ([]string, error) {
	existing := map[string]bool{}
	missingh := map[string]bool{}
	missing := []string{}

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)

		if len(parts) == 2 {
			existing[parts[0]] = true
		}
	}

	for _, entry := range *m {
		resolved, err := entry.ResolvedEnvironment(m)
		if err != nil {
			return missing, err
		}

		for _, env := range resolved {
			if strings.Index(env, "=") == -1 {
				if !existing[env] {
					missingh[env] = true
				}
			}
		}
	}

	for mm, _ := range missingh {
		missing = append(missing, mm)
	}

	sort.Strings(missing)

	return missing, nil
}

func (m *Manifest) PortConflicts() ([]string, error) {
	wanted := m.PortsWanted()

	conflicts := make([]string, 0)

	host := dockerHost()

	for _, p := range wanted {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, p), 200*time.Millisecond)

		if err == nil {
			conflicts = append(conflicts, p)
			defer conn.Close()
		}
	}

	return conflicts, nil
}

func (m *Manifest) PortsWanted() []string {
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

	return ports
}

func (m *Manifest) Push(app, registry, auth, tag string, flatten string) []error {
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

		if flatten != "" {
			remote = fmt.Sprintf("%s/%s:%s", registry, flatten, fmt.Sprintf("%s.%s", name, tag))
		}

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
		go (*m)[name].runAsync(m, m.prefixForEntry(name, i), app, name, ch)
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

func (me ManifestEntry) runAsync(m *Manifest, prefix, app, process string, ch chan error) {
	tag := fmt.Sprintf("%s/%s", app, process)
	name := containerName(app, process)

	query("docker", "rm", "-f", name)

	args := []string{"run", "-i", "--name", name}

	resolved, err := me.ResolvedEnvironment(m)

	if err != nil {
		ch <- err
		return
	}

	for _, env := range resolved {
		args = append(args, "-e", env)
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

func dockerHost() (host string) {
	host = "127.0.0.1"

	if h := os.Getenv("DOCKER_HOST"); h != "" {
		u, err := url.Parse(h)

		if err != nil {
			return
		}

		parts := strings.Split(u.Host, ":")
		host = parts[0]
	}

	return
}

// gets link entry env by pulling and inspecting the image for LINK_ vars
// overrides with vars specififed in the link's manifest
func getLinkEntryEnv(linkEntry ManifestEntry) (map[string]string, error) {
	linkEntryEnv := make(map[string]string)

	if linkEntry.Image != "" {
		pull := Execer("docker", "pull", linkEntry.Image)
		err := pull.Run()
		if err != nil {
			return linkEntryEnv, fmt.Errorf("could not pull container %q: %s", linkEntry.Image, err.Error())
		}

		cmd := Execer("docker", "inspect", linkEntry.Image)
		output, err := cmd.CombinedOutput()

		if err != nil {
			return linkEntryEnv, fmt.Errorf("could not inspect container %q: %s", linkEntry.Image, err.Error())
		}

		var inspect []struct {
			Config struct {
				Env []string
			}
		}

		err = json.Unmarshal(output, &inspect)
		if err != nil {
			return linkEntryEnv, err
		}

		if len(inspect) < 1 {
			return linkEntryEnv, fmt.Errorf("could not inspect container %q", linkEntry.Image)
		}

		for _, val := range inspect[0].Config.Env {
			parts := strings.SplitN(val, "=", 2)
			if len(parts) == 2 {
				linkEntryEnv[parts[0]] = parts[1]
			}
		}
	}

	//override with manifest env
	for _, value := range linkEntry.EnvironmentArray() {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) == 2 {
			linkEntryEnv[parts[0]] = parts[1]
		}
	}

	return linkEntryEnv, nil
}

// gets port from linkEntry's manifest
// throws error for no ports
// uses first port in the list otherwise
// uses exposed side of p1:p2 port mappings (p1)
func resolveOtherPort(name string, linkEntry ManifestEntry) string {
	var port string
	switch t := linkEntry.Ports.(type) {
	case []string:
		if len(t) < 1 {
			return ""
		}

		port = t[0]
	case []interface{}:
		if len(t) < 1 {
			return ""
		}

		port = fmt.Sprintf("%v", t[0])
	}

	port = strings.Split(port, ":")[0]
	return port
}

// gets ip address for docker gateway for network lookup
func getDockerGateway() (string, error) {
	cmd := Execer("docker", "run", "convox/docker-gateway")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	host := strings.TrimSpace(string(output))
	return host, nil
}
