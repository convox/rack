package manifest

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/convox/rack/cmd/convox/changes"
	"github.com/convox/rack/cmd/convox/templates"
	"github.com/docker/docker/builder/dockerignore"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/fatih/color"
	"github.com/fsouza/go-dockerclient"
	yaml "gopkg.in/yaml.v2"
)

//NOTE: these vars allow us to control other shell-outs during testing
var (
	Stdout       = io.Writer(os.Stdout)
	Stderr       = io.Writer(os.Stderr)
	Execer       = exec.Command
	SignalWaiter = waitForSignal

	regexValidProcessName = regexp.MustCompile(`\A[a-zA-Z0-9][-a-zA-Z0-9]{0,29}\z`) // 'web', '1', 'web-1' valid; '-', 'web_1' invalid
)

var (
	special = color.New(color.FgWhite).Add(color.Bold).SprintFunc()
	command = color.New(color.FgBlack).Add(color.Bold).SprintFunc()
	warning = color.New(color.FgYellow).Add(color.Bold).SprintFunc()
	system  = color.New(color.FgBlack).Add(color.Bold).SprintFunc()
)

// YAMLError is an error type to use when the user-supplied manifest YAML is not valid
// This can be treated as a validation error, not an exception
type YAMLError struct {
	err error
}

func (y *YAMLError) Error() string {
	return y.err.Error()
}

var RandomPort = func() int {
	return 10000 + rand.Intn(50000)
}

var Colors = []color.Attribute{color.FgCyan, color.FgYellow, color.FgGreen, color.FgMagenta, color.FgBlue}

type Manifest map[string]ManifestEntry

type Networks map[string]InternalNetwork

type InternalNetwork map[string]ExternalNetwork

type ExternalNetwork Network

type Network struct {
	Name string
}

type ManifestV2 struct {
	Version  string
	Networks Networks
	Services Manifest
}

type ManifestEntry struct {
	Build       string      `yaml:"build,omitempty"`
	Dockerfile  string      `yaml:"dockerfile,omitempty"`
	Image       string      `yaml:"image,omitempty"`
	Command     interface{} `yaml:"command,omitempty"`
	Entrypoint  string      `yaml:"entrypoint,omitempty"`
	Environment interface{} `yaml:"environment,omitempty"`
	Labels      interface{} `yaml:"labels,omitempty"`
	Links       []string    `yaml:"links,omitempty"`
	Networks    Networks    `yaml:"networks,omitempty"`
	Ports       interface{} `yaml:"ports,omitempty"`
	Privileged  bool        `yaml:"privileged,omitempty"`
	Volumes     []string    `yaml:"volumes,omitempty"`
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func Init(dir string) error {
	return initApplication(dir)
}

func Read(dir, filename string) (*Manifest, error) {
	data, err := ioutil.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return nil, fmt.Errorf("file not found: %s", filename)
	}

	var mv2 ManifestV2
	var m Manifest
	var n Networks

	err = yaml.Unmarshal(data, &mv2)
	if err != nil {
		return nil, &YAMLError{err}
	}

	if mv2.Version == "" {
		err = yaml.Unmarshal(data, &m)
		if err != nil {
			return nil, &YAMLError{err}
		}
	} else {
		m = mv2.Services
		n = mv2.Networks
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
		if !regexValidProcessName.MatchString(name) {
			return &m, fmt.Errorf("process name %q is invalid. It should contain only alphanumeric characters and dashes.", name)
		}

		for i, volume := range entry.Volumes {
			parts := strings.Split(volume, ":")

			for j, part := range parts {
				if !filepath.IsAbs(part) {
					parts[j] = filepath.Join(dir, part)
				}
			}

			entry.Volumes[i] = strings.Join(parts, ":")
		}

		entry.Networks = n
		m[name] = entry
	}

	err = m.Validate()
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// Validate convox-specific convox labels and values for every entry
func (m Manifest) Validate() error {
	regexValidCronLabel := regexp.MustCompile(`\A[a-zA-Z][-a-zA-Z0-9]{3,29}\z`)

	for _, entry := range map[string]ManifestEntry(m) {
		labels := entry.LabelsByPrefix("convox.cron")
		for k := range labels {
			parts := strings.Split(k, ".")
			if len(parts) != 3 {
				return fmt.Errorf(
					"Cron task is not valid (must be in format convox.cron.myjob)",
				)
			}
			name := parts[2]
			if !regexValidCronLabel.MatchString(name) {
				return fmt.Errorf(
					"Cron task %s is not valid (cron names can contain only alphanumeric characters and dashes and must be between 4 and 30 characters)",
					name,
				)
			}
		}
	}
	return nil
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
	err := run("docker", "tag", local, remote)

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
			err := Execer("docker", "inspect", entry.Image).Run()

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
		err := run("docker", "tag", from, to)

		if err != nil {
			return []error{err}
		}
	}

	return []error{}
}

func (me *ManifestEntry) ResolvedEnvironment(m *Manifest, cache bool, app string) ([]string, error) {
	r := []string{}

	linkedVars, err := me.ResolvedLinkVars(m, cache, app)
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
func (me *ManifestEntry) ResolvedLinkVars(m *Manifest, cache bool, app string) (map[string]string, error) {
	linkVars := make(map[string]string)

	if m == nil {
		return linkVars, nil
	}

	for _, link := range me.Links {
		linkEntry, ok := (*m)[link]

		if !ok {
			return nil, fmt.Errorf("no such link: %s", link)
		}

		linkEntryEnv, err := getLinkEntryEnv(linkEntry, cache)
		if err != nil {
			return linkVars, err
		}

		// get url parts from various places
		scheme := linkEntryEnv["LINK_SCHEME"]
		if scheme == "" {
			scheme = "tcp"
		}

		procName := fmt.Sprintf("%s-%s", app, link)
		// we don't create a balancer without a port,
		// so we don't create a link url either
		port := resolveContainerPort(link, linkEntry)
		if port == "" {
			continue
		}

		linkUrl := url.URL{
			Scheme: scheme,
			Host:   procName + ":" + port,
			Path:   linkEntryEnv["LINK_PATH"],
		}

		if linkEntryEnv["LINK_USERNAME"] != "" || linkEntryEnv["LINK_PASSWORD"] != "" {
			linkUrl.User = url.UserPassword(linkEntryEnv["LINK_USERNAME"], linkEntryEnv["LINK_PASSWORD"])
		}

		prefix := strings.ToUpper(link) + "_"
		prefix = strings.Replace(prefix, "-", "_", -1)
		linkVars[prefix+"URL"] = linkUrl.String()
		linkVars[prefix+"HOST"] = procName
		linkVars[prefix+"SCHEME"] = scheme
		linkVars[prefix+"PORT"] = port
		linkVars[prefix+"USERNAME"] = linkEntryEnv["LINK_USERNAME"]
		linkVars[prefix+"PASSWORD"] = linkEntryEnv["LINK_PASSWORD"]
		linkVars[prefix+"PATH"] = linkEntryEnv["LINK_PATH"]
	}

	return linkVars, nil
}

func (m *Manifest) MissingEnvironment(cache bool, app string) ([]string, error) {
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
		resolved, err := entry.ResolvedEnvironment(m, cache, app)
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

func (m *Manifest) PortConflicts(shift int) ([]string, error) {
	wanted := m.PortsWanted(shift)

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

func (m *Manifest) PortsWanted(shift int) []string {
	ports := make([]string, 0)

	for _, entry := range *m {
		if pp, ok := entry.Ports.([]interface{}); ok {
			for _, port := range pp {
				if p, ok := port.(string); ok {
					parts := strings.SplitN(p, ":", 2)

					if len(parts) == 2 {
						pi, _ := strconv.Atoi(parts[0])
						ports = append(ports, fmt.Sprintf("%d", pi+shift))
					}
				}
			}
		}
	}

	return ports
}

func (m *Manifest) Push(app, registry, tag string, flatten string) []error {
	if tag == "" {
		tag = "latest"
	}

	for name, _ := range *m {
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

func (m *Manifest) Raw() ([]byte, error) {
	return yaml.Marshal(m)
}

func (m *Manifest) Run(app string, cache, sync bool, shift int) []error {
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
		}
	}()

	for i, name := range m.runOrder() {
		sch := make(chan error)
		go (*m)[name].runAsync(m, m.prefixForEntry(name, i), app, name, cache, shift, ch, sch)
		<-sch // block until started successfully

		if sync {
			time.Sleep(1 * time.Second)
			m.sync(app, name)
		}

		// block until we can get the container IP
		for {
			host := containerName(app, name)
			ipFilterString := "{{ .NetworkSettings.IPAddress }}"

			//if using a custom network, only use the first network
			if (*m)[name].Networks != nil {
				for _, n := range (*m)[name].Networks {
					for _, in := range n {
						ipFilterString = "{{ .NetworkSettings.Networks." + in.Name + ".IPAddress }}"
						break
					}
					break
				}
			}

			ip, err := Execer("docker", "inspect", "-f", ipFilterString, host).Output()

			// dont block in test mode
			if os.Getenv("PROVIDER") == "test" {
				break
			}

			if (err == nil) && (net.ParseIP(strings.TrimSpace(string(ip))) != nil) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	if sync {
		go m.syncFiles()
		go m.syncBack()
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

func (m *Manifest) sync(app, process string) error {
	err := (*m)[process].syncAdds(app, process)

	if err != nil {
		return err
	}

	return nil
}

func (me ManifestEntry) copyChangesBinaryToContainer(app, process string) error {
	// only sync containers with a build directive
	if me.Build == "" {
		return nil
	}

	dc, _ := docker.NewClientFromEnv()

	var buf bytes.Buffer

	tgz := tar.NewWriter(&buf)

	data, err := changes.Asset("../changes/changes")

	if err != nil {
		return err
	}

	tgz.WriteHeader(&tar.Header{
		Name: "changes",
		Mode: 0755,
		Size: int64(len(data)),
	})

	tgz.Write(data)

	tgz.Close()

	err = dc.UploadToContainer(containerName(app, process), docker.UploadToContainerOptions{
		InputStream: &buf,
		Path:        "/",
	})

	if err != nil {
		return err
	}

	return nil
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

func (m *Manifest) systemPrefix() string {
	name := "convox"
	longest := len(name)

	for name, _ := range *m {
		if len(name) > longest {
			longest = len(name)
		}
	}

	return name + strings.Repeat(" ", longest-len(name)) + " |"
}

func (m *Manifest) prefixForEntry(name string, pos int) string {
	longest := 6 // (convox)

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

func (me ManifestEntry) runAsync(m *Manifest, prefix, app, process string, cache bool, shift int, ch chan error, sch chan error) {
	tag := fmt.Sprintf("%s/%s", app, process)
	name := containerName(app, process)

	query("docker", "rm", "-f", name)

	args := []string{"run", "-i", "--name", name}

	resolved, err := me.ResolvedEnvironment(m, cache, app)

	if err != nil {
		ch <- err
		return
	}

	for _, env := range resolved {
		args = append(args, "-e", env)
	}

	if me.Privileged {
		args = append(args, "--privileged")
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

	if s := me.Label("convox.start.shift"); s != "" {
		si, err := strconv.Atoi(s)

		if err != nil {
			ch <- err
			return
		}

		shift = si
	}

	host := ""
	container := ""

	for _, port := range ports {
		switch len(strings.Split(port, ":")) {
		case 1:
			host = port
			container = port
		case 2:
			parts := strings.SplitN(port, ":", 2)
			host = parts[0]
			container = parts[1]
		default:
			ch <- fmt.Errorf("unknown port declaration: %s", port)
			return
		}

		hosti, err := strconv.Atoi(host)

		if err != nil {
			ch <- err
			return
		}

		host = fmt.Sprintf("%d", hosti+shift)

		switch proto := me.Label(fmt.Sprintf("convox.port.%s.protocol", host)); proto {
		case "https", "tls":
			proxy := false
			secure := false

			if me.Label(fmt.Sprintf("convox.port.%s.proxy", host)) == "true" {
				proxy = true
			}

			if me.Label(fmt.Sprintf("convox.port.%s.secure", host)) == "true" {
				secure = true
			}

			rnd := RandomPort()
			fmt.Println(prefix, special(fmt.Sprintf("%s proxy enabled for %s:%s", proto, host, container)))
			go proxyPort(proto, host, container, name, proxy, secure)
			host = strconv.Itoa(rnd)
		}

		args = append(args, "-p", fmt.Sprintf("%s:%s", host, container))
	}

	for _, volume := range me.Volumes {
		warnIfRoot(volume)
		args = append(args, "-v", volume)
	}

	if me.Entrypoint != "" {
		args = append(args, "--entrypoint", me.Entrypoint)
	}

	if me.Networks != nil {
		for _, n := range me.Networks {
			for _, in := range n {
				args = append(args, "--net", in.Name)
			}
		}
	} else {
		// add link container hostnames to the /etc/hosts of each run
		for _, link := range me.Links {
			host := containerName(app, link)

			ip, err := Execer("docker", "inspect", "-f", "{{ .NetworkSettings.IPAddress }}", host).CombinedOutput()

			if err != nil {
				ch <- err
				return
			}

			// Add the convox container name to the host list which is app + link
			args = append(args, fmt.Sprintf(`--add-host=%s:%s`, host, strings.TrimSpace(string(ip))))

			// Add traditional docker-compose host link which is simply container name, in this case it's called link
			args = append(args, fmt.Sprintf(`--add-host=%s:%s`, link, strings.TrimSpace(string(ip))))
		}
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

	ch <- runPrefix(prefix, sch, "docker", args...)
}

func (me ManifestEntry) Label(key string) string {
	switch labels := me.Labels.(type) {
	case map[interface{}]interface{}:
		for k, v := range labels {
			ks, ok := k.(string)

			if !ok {
				return ""
			}

			vs, ok := v.(string)

			if !ok {
				return ""
			}

			if ks == key {
				return vs
			}
		}
	case []interface{}:
		for _, label := range labels {
			ls, ok := label.(string)

			if !ok {
				return ""
			}

			if parts := strings.SplitN(ls, "=", 2); len(parts) == 2 {
				if parts[0] == key {
					return parts[1]
				}
			}
		}
	}

	return ""
}

// LabelsByPrefix filters Docker Compose label names by prefixes and returns
// a map of label names to values that match.
func (e ManifestEntry) LabelsByPrefix(prefix string) map[string]string {
	returnLabels := make(map[string]string)
	switch labels := e.Labels.(type) {
	case map[interface{}]interface{}:
		for k, v := range labels {
			ks, ok := k.(string)

			if !ok {
				continue
			}

			vs, ok := v.(string)

			if !ok {
				continue
			}

			if strings.HasPrefix(ks, prefix) {
				returnLabels[ks] = vs
			}
		}
	case []interface{}:
		for _, label := range labels {
			ls, ok := label.(string)

			if !ok {
				continue
			}

			if parts := strings.SplitN(ls, "=", 2); len(parts) == 2 {
				if strings.HasPrefix(parts[0], prefix) {
					returnLabels[parts[0]] = parts[1]
				}
			}
		}
	}
	return returnLabels
}

func (me ManifestEntry) Protocol(port string) string {
	proto := "tcp"

	if p := me.Label(fmt.Sprintf("com.convox.port.%s.protocol", port)); p != "" {
		proto = p
	}

	return proto
}

func (me ManifestEntry) syncAdds(app, process string) error {
	// only sync containers with a build directive
	if me.Build == "" {
		return nil
	}

	err := me.copyChangesBinaryToContainer(app, process)

	if err != nil {
		return err
	}

	dockerfile := filepath.Join(me.Build, "Dockerfile")

	if me.Dockerfile != "" {
		dockerfile = me.Dockerfile
	}

	data, err := ioutil.ReadFile(dockerfile)

	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)

		if len(parts) < 1 {
			continue
		}

		switch parts[0] {
		case "ADD", "COPY":
			if len(parts) < 3 {
				continue
			}

			remote := parts[2]

			if !filepath.IsAbs(remote) {
				wdb, err := Execer("docker", "inspect", "-f", "{{.Config.WorkingDir}}", containerName(app, process)).Output()
				if err != nil {
					return err
				}

				wd := strings.TrimSpace(string(wdb))

				remote = filepath.Join(wd, remote)
			}

			local := filepath.Join(me.Build, parts[1])

			info, err := os.Stat(local)
			if err != nil {
				return err
			}

			if !info.IsDir() && strings.HasSuffix(remote, "/") {
				remote = filepath.Join(remote, filepath.Base(local))
			}

			registerSync(containerName(app, process), filepath.Join(me.Build, parts[1]), remote)
		}
	}

	return nil
}

func exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

func proxyPort(protocol, from, to, link string, proxy, secure bool) {
	args := []string{"run", "-p", fmt.Sprintf("%s:%s", from, from), "--link", fmt.Sprintf("%s:host", link), "convox/proxy", from, to, protocol}

	if proxy {
		args = append(args, "proxy")
	}

	if secure {
		args = append(args, "secure")
	}

	// wait for main container to come up
	time.Sleep(1 * time.Second)

	cmd := Execer("docker", args...)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	cmd.Run()
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

func runPrefix(prefix string, sch chan error, executable string, args ...string) error {
	fmt.Println(prefix, command(fmt.Sprintf("%s %s", executable, strings.Join(args, " "))))

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

	sch <- nil // signal that start worked

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

func randomString(prefix string, size int) string {
	b := make([]rune, size)
	for i := range b {
		b[i] = randomAlphabet[rand.Intn(len(randomAlphabet))]
	}
	return prefix + string(b)
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
func getLinkEntryEnv(linkEntry ManifestEntry, cache bool) (map[string]string, error) {
	linkEntryEnv := make(map[string]string)

	if linkEntry.Image != "" {
		err := Execer("docker", "inspect", linkEntry.Image).Run()

		if err != nil || !cache {
			pull := Execer("docker", "pull", linkEntry.Image)
			err := pull.Run()
			if err != nil {
				return linkEntryEnv, fmt.Errorf("could not pull container %q: %s", linkEntry.Image, err.Error())
			}
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
func resolveContainerPort(name string, linkEntry ManifestEntry) string {
	var port string
	switch t := linkEntry.Ports.(type) {
	case []string:
		if len(t) < 1 {
			return ""
		}

		port = t[1]
	case []interface{}:
		if len(t) < 1 {
			return ""
		}
		port = fmt.Sprintf("%v", t[0])
	}

	ports := strings.Split(port, ":")

	if len(ports) == 1 {
		return ports[0]
	}

	return ports[1]
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

type Files map[string]time.Time

func watchWalker(files Files, local string, adds map[string]bool, lock sync.Mutex, basePath string, patterns []string, patDirs [][]string, exceptions bool) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}

		skip, err := fileutils.OptimizedMatches(relPath, patterns, patDirs)
		if err != nil {
			log.Printf("Error matching %s: %v", relPath, err)
			return err
		}
		if skip {
			return nil
		}

		if files[path] != info.ModTime() {
			files[path] = info.ModTime()

			blockLocalAddsLock.Lock()

			if blockLocalAdds[path] > 0 {
				blockLocalAdds[path] -= 1
			} else {
				lock.Lock()
				adds[path] = true
				lock.Unlock()
			}

			blockLocalAddsLock.Unlock()
		}

		return nil
	}
}

func processAdds(prefix string, adds map[string]bool, lock sync.Mutex, syncs []Sync) {
	dc, _ := docker.NewClientFromEnv()

	for {
		time.Sleep(1 * time.Second)

		if len(adds) == 0 {
			continue
		}

		lock.Lock()

		fmt.Printf(system("%s uploading %d files\n"), prefix, len(adds))

		if os.Getenv("CONVOX_DEBUG") != "" {
			for add := range adds {
				fmt.Printf(system("%s uploading %s\n"), prefix, add)
			}
		}

		for _, sync := range syncs {
			var buf bytes.Buffer

			tgz := tar.NewWriter(&buf)

			for local := range adds {
				info, err := os.Stat(local)

				if err != nil {
					continue
				}

				rel, err := filepath.Rel(sync.Local, local)
				if err != nil {
					continue
				}

				remote := filepath.Join(sync.Remote, rel)

				blockRemoteAddsLock.Lock()
				blockRemoteAdds[remote] += 1
				blockRemoteAddsLock.Unlock()

				tgz.WriteHeader(&tar.Header{
					Name:    remote,
					Mode:    0644,
					Size:    info.Size(),
					ModTime: info.ModTime(),
				})

				fd, err := os.Open(local)

				if err != nil {
					continue
				}

				io.Copy(tgz, fd)
				fd.Close()
			}

			tgz.Close()

			err := dc.UploadToContainer(sync.Container, docker.UploadToContainerOptions{
				InputStream: &buf,
				Path:        "/",
			})

			if err != nil {
				fmt.Printf("err: %+v\n", err)
				continue
			}
		}

		for key := range adds {
			delete(adds, key)
		}

		lock.Unlock()
	}
}

func processRemoves(prefix string, removes map[string]bool, lock sync.Mutex, syncs []Sync) {
	dc, _ := docker.NewClientFromEnv()

	for {
		time.Sleep(1 * time.Second)

		if len(removes) == 0 {
			continue
		}

		cmd := []string{"rm", "-f"}

		lock.Lock()

		fmt.Printf(system(fmt.Sprintf("%s removing %d remote files\n", prefix, len(removes))))

		for file := range removes {
			cmd = append(cmd, file)
			delete(removes, file)
		}

		lock.Unlock()

		for _, sync := range syncs {
			res, err := dc.CreateExec(docker.CreateExecOptions{
				Container: sync.Container,
				Cmd:       cmd,
			})

			if err != nil {
				fmt.Printf("err: %+v\n", err)
				continue
			}

			err = dc.StartExec(res.ID, docker.StartExecOptions{
				Detach: true,
			})

			if err != nil {
				fmt.Printf("err: %+v\n", err)
				continue
			}
		}

		for key := range removes {
			delete(removes, key)
		}
	}
}

type Sync struct {
	Container string
	Local     string
	Remote    string
}

var syncs = []Sync{}

func registerSync(container, local, remote string) error {
	syscall.Setrlimit(syscall.RLIMIT_NOFILE, &syscall.Rlimit{
		Max: 999999,
		Cur: 999999,
	})

	abs, err := filepath.Abs(local)

	if err != nil {
		return err
	}

	sym, err := filepath.EvalSymlinks(abs)

	if err != nil {
		return err
	}

	syncs = append(syncs, Sync{
		Container: container,
		Local:     sym,
		Remote:    remote,
	})

	return nil
}

func (m *Manifest) syncBack() error {
	containers := map[string][]string{}

	// Group sync directories by container
	for _, sync := range syncs {
		containers[sync.Container] = append(containers[sync.Container], sync.Remote)
	}

	dc, _ := docker.NewClientFromEnv()

	for container, dirs := range containers {
		exec, err := dc.CreateExec(docker.CreateExecOptions{
			AttachStdin:  false,
			AttachStdout: true,
			AttachStderr: true,
			Cmd:          append([]string{"/changes"}, dirs...),
			Container:    container,
		})

		if err != nil {
			fmt.Printf("err = %+v\n", err)
			return err
		}

		r, w := io.Pipe()

		go m.scanRemoteChanges(container, r)
		go m.processRemoteChanges(container)

		err = dc.StartExec(exec.ID, docker.StartExecOptions{
			Tty:          true,
			RawTerminal:  true,
			OutputStream: w,
			ErrorStream:  w,
		})

		fmt.Println("terminated")

		if err != nil {
			fmt.Printf("err = %+v\n", err)
			return err
		}

	}

	return nil
}

var (
	remoteAdds          = map[string]string{}
	remoteRemoves       = map[string]string{}
	remoteAddLock       sync.Mutex
	remoteRemoveLock    sync.Mutex
	blockLocalAdds      = map[string]int{}
	blockLocalAddsLock  sync.Mutex
	blockRemoteAdds     = map[string]int{}
	blockRemoteAddsLock sync.Mutex
)

func (m *Manifest) scanRemoteChanges(container string, r io.Reader) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "|")

		if len(parts) == 3 {
			for _, sync := range syncs {
				local := filepath.Join(sync.Local, parts[2])
				remote := filepath.Join(parts[1], parts[2])

				if sync.Remote == parts[1] {
					switch parts[0] {
					case "add":
						blockRemoteAddsLock.Lock()

						if blockRemoteAdds[remote] > 0 {
							blockRemoteAdds[remote] -= 1
						} else {
							remoteAddLock.Lock()
							remoteAdds[local] = remote
							remoteAddLock.Unlock()
						}

						blockRemoteAddsLock.Unlock()
					case "delete":
						// remoteRemoveLock.Lock()
						// remoteRemoves[filepath.Join(sync.Local, parts[2])] = filepath.Join(parts[1], parts[2])
						// remoteRemoveLock.Unlock()
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("scanner error")
	}
}

func (m *Manifest) processRemoteChanges(container string) {
	for {
		remoteAddLock.Lock()

		num := 0
		adds := map[string]string{}

		for local, remote := range remoteAdds {
			adds[local] = remote
			delete(remoteAdds, local)
			num += 1

			if num >= 1000 {
				break
			}
		}

		remoteAddLock.Unlock()

		go m.downloadRemoteAdds(container, adds)

		remoteRemoveLock.Lock()

		if len(remoteRemoves) > 0 {
			fmt.Printf(system("%s removing %d local files\n"), m.systemPrefix(), len(remoteRemoves))

			for local, _ := range remoteRemoves {
				os.Remove(local)
				delete(remoteRemoves, local)
			}
		}

		remoteRemoveLock.Unlock()

		time.Sleep(1 * time.Second)
	}
}

func (m *Manifest) downloadRemoteAdds(container string, adds map[string]string) {
	if len(adds) == 0 {
		return
	}

	remotes := []string{}
	locals := map[string]string{}

	for local, remote := range adds {
		remotes = append(remotes, remote)
		locals[remote] = local
	}

	fmt.Printf(system("%s downloading %d files\n"), m.systemPrefix(), len(remotes))

	if os.Getenv("CONVOX_DEBUG") != "" {
		for _, remote := range remotes {
			fmt.Printf(system("%s downloading %s\n"), m.systemPrefix(), remote)
		}
	}

	args := append([]string{"exec", "-i", container, "tar", "czf", "-"}, remotes...)

	data, err := Execer("docker", args...).Output()

	if err != nil {
		fmt.Printf("err = %+v\n", err)
		return
	}

	gz, err := gzip.NewReader(bytes.NewBuffer(data))

	if err != nil {
		fmt.Printf("err = %+v\n", err)
		return
	}

	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			fmt.Printf("err = %+v\n", err)
			continue
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if local, ok := locals[filepath.Join("/", header.Name)]; ok {
				if err := os.MkdirAll(filepath.Dir(local), 0755); err != nil {
					fmt.Printf("err = %+v\n", err)
					continue
				}

				if _, err := os.Stat(local); !os.IsNotExist(err) {
					if err := os.Remove(local); err != nil {
						fmt.Printf("err = %+v\n", err)
						continue
					}
				}

				blockLocalAddsLock.Lock()
				blockLocalAdds[local] += 1
				blockLocalAddsLock.Unlock()

				fd, err := os.OpenFile(local, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))

				if err != nil {
					fmt.Printf("err = %+v\n", err)
					continue
				}

				defer fd.Close()

				_, err = io.Copy(fd, tr)

				if err != nil {
					fmt.Printf("err = %+v\n", err)
					continue
				}
			}
		}
	}
}

func (m *Manifest) syncFiles() error {
	watches := map[string][]Sync{}
	candidates := []string{}

	for _, sync := range syncs {
		candidates = append(candidates, sync.Local)
	}

	sort.Strings(candidates)

	for _, candidate := range candidates {
		contained := false

		for watch := range watches {
			if strings.HasPrefix(candidate, watch) {
				contained = true
				break
			}
		}

		if !contained {
			watches[candidate] = []Sync{}
		}
	}

	for _, sync := range syncs {
		for watch := range watches {
			if sync.Local == watch {
				watches[watch] = append(watches[watch], sync)
			}
		}
	}

	for watch, syncs := range watches {
		go m.processSync(watch, syncs)
	}

	return nil
}

func (m *Manifest) processSync(local string, syncs []Sync) error {
	files := Files{}

	adds := map[string]bool{}
	removes := map[string]bool{}

	var alock, rlock sync.Mutex

	go processAdds(m.systemPrefix(), adds, alock, syncs)
	go processRemoves(m.systemPrefix(), removes, rlock, syncs)

	filepath.Walk(local, func(path string, info os.FileInfo, err error) error {
		if info != nil {
			files[path] = info.ModTime()
		}
		return nil
	})

	for {
		di, err := readDockerIgnore()
		if err != nil {
			return err
		}

		patterns, patDirs, exceptions, err := fileutils.CleanPatterns(di)
		if err != nil {
			return err
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		err = filepath.Walk(local, watchWalker(files, local, adds, alock, cwd, patterns, patDirs, exceptions))
		for _, sync := range syncs {
			if err != nil {
				fmt.Printf("err: %+v\n", err)
				continue
			}

			for file := range files {
				if _, err := os.Stat(file); os.IsNotExist(err) {
					rel, err := filepath.Rel(local, file)

					if err != nil {
						continue
					}

					rlock.Lock()
					removes[filepath.Join(sync.Remote, rel)] = true
					rlock.Unlock()

					delete(files, file)
				}
			}
		}

		time.Sleep(900 * time.Millisecond)
	}

	return nil
}

func warnIfRoot(volume string) {
	resv, _ := filepath.EvalSymlinks(volume)
	absv, _ := filepath.Abs(resv)
	wd, _ := os.Getwd()

	if absv == wd {
		fmt.Println(warning("WARNING: Detected application directory mounted as volume"))
		fmt.Println(warning("convox start will automatically synchronize any files referenced by ADD or COPY statements"))
	}
}

// init

func detectApplication(dir string) string {
	switch {
	// case exists(filepath.Join(dir, ".meteor")):
	//   return "meteor"
	// case exists(filepath.Join(dir, "package.json")):
	//   return "node"
	case exists(filepath.Join(dir, "config/application.rb")):
		return "rails"
	case exists(filepath.Join(dir, "config.ru")):
		return "sinatra"
	case exists(filepath.Join(dir, "Gemfile.lock")):
		return "ruby"
	}

	return "unknown"
}

func initApplication(dir string) error {
	wd, err := os.Getwd()

	if err != nil {
		return err
	}

	defer os.Chdir(wd)

	os.Chdir(dir)

	if exists("docker-compose.yml") {
		return nil
	}

	kind := detectApplication(dir)

	fmt.Printf("Initializing %s\n", kind)

	// TODO parse the Dockerfile and build a docker-compose.yml
	if !exists("Dockerfile") {
		if err := writeAsset("Dockerfile", fmt.Sprintf("init/%s/Dockerfile", kind)); err != nil {
			return err
		}
	}

	if err := generateManifest(dir, fmt.Sprintf("init/%s/docker-compose.yml", kind)); err != nil {
		return err
	}

	if err := writeAsset(".dockerignore", fmt.Sprintf("init/%s/.dockerignore", kind)); err != nil {
		return err
	}

	return nil
}

func generateManifest(dir string, def string) error {
	if exists("Procfile") {
		pf, err := readProcfile("Procfile")

		if err != nil {
			return err
		}

		m := Manifest{}

		for _, e := range pf {
			me := ManifestEntry{
				Build:   ".",
				Command: e.Command,
			}

			switch e.Name {
			case "web":
				me.Labels = []string{
					"convox.port.443.protocol=tls",
					"convox.port.443.proxy=true",
				}

				me.Ports = []string{
					"80:4000",
					"443:4001",
				}
			}

			m[e.Name] = me
		}

		data, err := yaml.Marshal(m)

		if err != nil {
			return err
		}

		// write the generated docker-compose.yml and return
		return writeFile("docker-compose.yml", data, 0644)
	}

	// write the default if we get here
	return writeAsset("docker-compose.yml", def)
}

type ProcfileEntry struct {
	Name    string
	Command string
}

type Procfile []ProcfileEntry

var reProcfile = regexp.MustCompile("^([A-Za-z0-9_]+):\\s*(.+)$")

func readProcfile(path string) (Procfile, error) {
	data, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, err
	}

	pf := Procfile{}

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		parts := reProcfile.FindStringSubmatch(scanner.Text())

		if len(parts) == 3 {
			pf = append(pf, ProcfileEntry{
				Name:    parts[1],
				Command: parts[2],
			})
		}
	}

	return pf, nil
}

func writeFile(path string, data []byte, mode os.FileMode) error {
	fmt.Printf("Writing %s... ", path)

	if exists(path) {
		fmt.Println("EXISTS")
		return nil
	}

	// make the containing directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(path, data, mode); err != nil {
		return err
	}

	fmt.Println("OK")

	return nil
}

func writeAsset(path, template string) error {
	data, err := templates.Asset(template)

	if err != nil {
		return err
	}

	info, err := templates.AssetInfo(template)

	if err != nil {
		return err
	}

	return writeFile(path, data, info.Mode())
}

func readDockerIgnore() ([]string, error) {
	fd, err := os.Open(".dockerignore")
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}

	ignore, err := dockerignore.ReadAll(fd)
	if err != nil {
		return nil, err
	}

	return ignore, nil
}
