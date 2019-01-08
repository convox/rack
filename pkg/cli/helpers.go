package cli

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/convox/rack/pkg/options"
	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/convox/stdcli"
	"github.com/convox/stdsdk"
)

type rack struct {
	Name   string
	Status string
}

func app(c *stdcli.Context) string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return coalesce(c.String("app"), c.LocalSetting("app"), filepath.Base(wd))
}

func coalesce(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}

	return ""
}

func copySystemLogs(ctx context.Context, w io.Writer, r io.Reader) {
	s := bufio.NewScanner(r)

	for s.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		parts := strings.SplitN(s.Text(), " ", 3)

		if len(parts) < 3 {
			continue
		}

		if strings.HasPrefix(parts[1], "system/aws") {
			w.Write([]byte(fmt.Sprintf("%s\n", s.Text())))
		}
	}
}

func currentHost(c *stdcli.Context) (string, error) {
	if h := os.Getenv("CONVOX_HOST"); h != "" {
		return h, nil
	}

	if h, _ := c.SettingRead("host"); h != "" {
		return h, nil
	}

	return "", nil
}

func currentPassword(c *stdcli.Context, host string) (string, error) {
	if pw := os.Getenv("CONVOX_PASSWORD"); pw != "" {
		return pw, nil
	}

	return c.SettingReadKey("auth", host)
}

func currentEndpoint(c *stdcli.Context, rack_ string) (string, error) {
	if e := os.Getenv("RACK_URL"); e != "" {
		return e, nil
	}

	if strings.HasPrefix(rack_, "local/") {
		return fmt.Sprintf("https://rack.%s", strings.SplitN(rack_, "/", 2)[1]), nil
	}

	host, err := currentHost(c)
	if err != nil {
		return "", err
	}

	if host == "" {
		if !localRackRunning(c) {
			return "", fmt.Errorf("no racks found, try `convox login`")
		}

		var r *rack

		if cr := currentRack(c, ""); cr != "" {
			r, err = matchRack(c, cr)
			if err != nil {
				return "", err
			}
		} else {
			r, err = matchRack(c, "local/")
			if err != nil {
				return "", err
			}
		}

		if r == nil {
			return "", fmt.Errorf("no racks found, try `convox login`")
		}

		return fmt.Sprintf("https://rack.%s", strings.SplitN(r.Name, "/", 2)[1]), nil
	}

	pw, err := currentPassword(c, host)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://convox:%s@%s", url.QueryEscape(pw), host), nil
}

func currentRack(c *stdcli.Context, host string) string {
	if r := c.String("rack"); r != "" {
		return r
	}

	if r := os.Getenv("CONVOX_RACK"); r != "" {
		return r
	}

	if r := c.LocalSetting("rack"); r != "" {
		return r
	}

	if r := hostRacks(c)[host]; r != "" {
		return r
	}

	if r, _ := c.SettingRead("rack"); r != "" {
		return r
	}

	return ""
}

func executableName() string {
	switch runtime.GOOS {
	case "windows":
		return "convox.exe"
	default:
		return "convox"
	}
}

func generateTempKey() (string, error) {
	data := make([]byte, 1024)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)

	return fmt.Sprintf("tmp/%s", hex.EncodeToString(hash[:])[0:30]), nil
}

func handleSignalTermination(c *stdcli.Context, name string) {
	sigs := make(chan os.Signal)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for range sigs {
		fmt.Printf("\nstopping: %s\n", name)
		c.Run("docker", "stop", name)
	}
}

func hostRacks(c *stdcli.Context) map[string]string {
	data, err := c.SettingRead("racks")
	if err != nil {
		return map[string]string{}
	}

	var rs map[string]string

	if err := json.Unmarshal([]byte(data), &rs); err != nil {
		return map[string]string{}
	}

	return rs
}

func localRackRunning(c *stdcli.Context) bool {
	rs, err := localRacks(c)
	if err != nil {
		return false
	}

	return len(rs) > 0
}

func localRacks(c *stdcli.Context) ([]rack, error) {
	racks := []rack{}

	data, err := c.Execute("docker", "ps", "--filter", "label=convox.type=rack", "--format", "{{.Names}}")
	if err != nil {
		return []rack{}, nil // if no docker then no local racks
	}

	names := strings.Split(strings.TrimSpace(string(data)), "\n")

	for _, name := range names {
		if name == "" {
			continue
		}

		racks = append(racks, rack{
			Name:   fmt.Sprintf("local/%s", name),
			Status: "running",
		})
	}

	data, err = c.Execute("kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name")
	if err == nil {
		nsrs := strings.Split(strings.TrimSpace(string(data)), "\n")

		for _, nsr := range nsrs {
			if strings.HasPrefix(nsr, "namespace/") {
				racks = append(racks, rack{
					Name:   fmt.Sprintf("local/%s", strings.TrimPrefix(nsr, "namespace/")),
					Status: "running",
				})
			}
		}
	}

	return racks, nil
}

func matchRack(c *stdcli.Context, name string) (*rack, error) {
	rs, err := racks(c)
	if err != nil {
		return nil, err
	}

	matches := []rack{}

	for _, r := range rs {
		if r.Name == name {
			return &r, nil
		}

		if strings.Index(r.Name, name) != -1 {
			matches = append(matches, r)
		}
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous rack name: %s", name)
	}

	if len(matches) == 1 {
		return &matches[0], nil
	}

	return nil, fmt.Errorf("could not find rack: %s", name)
}

func rackCommand(name string, version string, router string, id string) (string, []string, error) {
	vol := "/var/convox"

	switch runtime.GOOS {
	case "darwin":
		vol = "/Users/Shared/convox"
	}

	image := fmt.Sprintf("convox/rack:%s", version)

	args := []string{"run", "--rm"}
	args = append(args, "-e", "COMBINED=true")
	args = append(args, "-e", fmt.Sprintf("ID=%s", id))
	args = append(args, "-e", fmt.Sprintf("IMAGE=%s", image))
	args = append(args, "-e", "PROVIDER=local")
	args = append(args, "-e", fmt.Sprintf("RACK=%s", name))
	args = append(args, "-e", fmt.Sprintf("ROUTER=%s", router))
	args = append(args, "-e", fmt.Sprintf("VERSION=%s", version))
	args = append(args, "-e", fmt.Sprintf("VOLUME=%s", vol))
	args = append(args, "-i")
	args = append(args, "--label", fmt.Sprintf("convox.rack=%s", name))
	args = append(args, "--label", "convox.type=rack")
	args = append(args, "-m", "256m")
	args = append(args, "--name", name)
	args = append(args, "-p", "5443")
	args = append(args, "-v", fmt.Sprintf("%s:/var/convox", vol))
	args = append(args, "-v", "/var/run/docker.sock:/var/run/docker.sock")
	args = append(args, image)

	return "docker", args, nil
}

func racks(c *stdcli.Context) ([]rack, error) {
	rs := []rack{}

	rrs, err := remoteRacks(c)
	if err != nil {
		return nil, err
	}

	rs = append(rs, rrs...)

	lrs, err := localRacks(c)
	if err != nil {
		return nil, err
	}

	rs = append(rs, lrs...)

	sort.Slice(rs, func(i, j int) bool {
		return rs[i].Name < rs[j].Name
	})

	return rs, nil
}

func remoteRacks(c *stdcli.Context) ([]rack, error) {
	h, err := currentHost(c)
	if err != nil {
		return nil, err
	}

	if h == "" {
		return []rack{}, nil
	}

	racks := []rack{}

	var rs []struct {
		Name         string
		Organization struct {
			Name string
		}
		Status string
	}

	// override local rack to get remote rack list
	endpoint, err := currentEndpoint(c, "")
	if err != nil {
		return nil, err
	}

	p, err := sdk.New(endpoint)
	if err != nil {
		return nil, err
	}

	p.Authenticator = authenticator(c)
	p.Session = currentSession(c)

	p.Get("/racks", stdsdk.RequestOptions{}, &rs)

	if rs != nil {
		for _, r := range rs {
			racks = append(racks, rack{
				Name:   fmt.Sprintf("%s/%s", r.Organization.Name, r.Name),
				Status: r.Status,
			})
		}
	}

	return racks, nil
}

func streamAppLogs(ctx context.Context, rack sdk.Interface, c *stdcli.Context, app string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		r, err := rack.AppLogs(app, structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(1)})
		if err != nil {
			return
		}

		copySystemLogs(ctx, c, r)

		time.Sleep(1 * time.Second)
	}
}

func streamRackSystemLogs(ctx context.Context, rack sdk.Interface, c *stdcli.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		r, err := rack.SystemLogs(structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(1)})
		if err != nil {
			return
		}

		copySystemLogs(ctx, c, r)

		time.Sleep(1 * time.Second)
	}
}

func tag(name, value string) string {
	return fmt.Sprintf("<%s>%s</%s>", name, value, name)
}

func wait(interval time.Duration, timeout time.Duration, times int, fn func() (bool, error)) error {
	successes := 0
	errors := 0
	start := time.Now().UTC()

	for {
		if start.Add(timeout).Before(time.Now().UTC()) {
			return fmt.Errorf("timeout")
		}

		success, err := fn()
		if err != nil {
			errors += 1
		} else {
			errors = 0
		}

		if errors >= times {
			return err
		}

		if success {
			successes += 1
		} else {
			successes = 0
		}

		if successes >= times {
			return nil
		}

		time.Sleep(interval)
	}
}

func waitForAppDeleted(rack sdk.Interface, c *stdcli.Context, app string) error {
	time.Sleep(WaitDuration) // give the stack time to start updating

	return wait(WaitDuration, 30*time.Minute, 2, func() (bool, error) {
		_, err := rack.AppGet(app)
		if err == nil {
			return false, nil
		}
		if strings.Contains(err.Error(), "no such app") {
			return true, nil
		}
		if strings.Contains(err.Error(), "app not found") {
			return true, nil
		}
		return false, err
	})
}

func waitForAppRunning(rack sdk.Interface, app string) error {
	time.Sleep(WaitDuration) // give the stack time to start updating

	var waitError error

	return wait(WaitDuration, 30*time.Minute, 2, func() (bool, error) {
		a, err := rack.AppGet(app)
		if err != nil {
			return false, err
		}

		if a.Status == "rollback" {
			waitError = fmt.Errorf("rollback")
		}

		return a.Status == "running", waitError
	})
}

func waitForAppWithLogs(rack sdk.Interface, c *stdcli.Context, app string) error {
	c.Writef("\n")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go streamAppLogs(ctx, rack, c, app)

	if err := waitForAppRunning(rack, app); err != nil {
		return err
	}

	return nil
}

func waitForProcessRunning(rack sdk.Interface, c *stdcli.Context, app, pid string) error {
	return wait(1*time.Second, 5*time.Minute, 2, func() (bool, error) {
		ps, err := rack.ProcessGet(app, pid)
		if err != nil {
			return false, err
		}

		return ps.Status == "running", nil
	})
}

func waitForRackRunning(rack sdk.Interface, c *stdcli.Context) error {
	time.Sleep(WaitDuration) // give the stack time to start updating

	return wait(WaitDuration, 30*time.Minute, 2, func() (bool, error) {
		s, err := rack.SystemGet()
		if err != nil {
			return false, err
		}

		return s.Status == "running", nil
	})
}

func waitForRackWithLogs(rack sdk.Interface, c *stdcli.Context) error {
	c.Writef("\n")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go streamRackSystemLogs(ctx, rack, c)

	if err := waitForRackRunning(rack, c); err != nil {
		return err
	}

	return nil
}

func waitForResourceDeleted(rack sdk.Interface, c *stdcli.Context, resource string) error {
	time.Sleep(WaitDuration) // give the stack time to start updating

	return wait(WaitDuration, 30*time.Minute, 2, func() (bool, error) {
		_, err := rack.ResourceGet(resource)
		if err == nil {
			return false, nil
		}
		if strings.Contains(err.Error(), "no such resource") {
			return true, nil
		}
		if strings.Contains(err.Error(), "does not exist") {
			return true, nil
		}
		return false, err
	})
}

func waitForResourceRunning(rack sdk.Interface, c *stdcli.Context, resource string) error {
	time.Sleep(WaitDuration) // give the stack time to start updating

	return wait(WaitDuration, 30*time.Minute, 2, func() (bool, error) {
		r, err := rack.ResourceGet(resource)
		if err != nil {
			return false, err
		}

		return r.Status == "running", nil
	})
}
