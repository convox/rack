package cli

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/convox/rack/pkg/helpers"
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

	data, err := c.Execute("kubectl", "get", "ns", "--selector=system=convox,type=rack", "--output=name")
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

func tag(name, value string) string {
	return fmt.Sprintf("<%s>%s</%s>", name, value, name)
}

func waitForResourceDeleted(rack sdk.Interface, c *stdcli.Context, resource string) error {
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	time.Sleep(WaitDuration) // give the stack time to start updating

	return helpers.Wait(WaitDuration, 30*time.Minute, 2, func() (bool, error) {
		var err error
		if s.Version <= "20190111211123" {
			_, err = rack.SystemResourceGetClassic(resource)
		} else {
			_, err = rack.SystemResourceGet(resource)
		}
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
	s, err := rack.SystemGet()
	if err != nil {
		return err
	}

	time.Sleep(WaitDuration) // give the stack time to start updating

	return helpers.Wait(WaitDuration, 30*time.Minute, 2, func() (bool, error) {
		var r *structs.Resource
		var err error

		if s.Version <= "20190111211123" {
			r, err = rack.SystemResourceGetClassic(resource)
		} else {
			r, err = rack.SystemResourceGet(resource)
		}
		if err != nil {
			return false, err
		}

		return r.Status == "running", nil
	})
}
