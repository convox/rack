package local

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"text/template"
	"time"

	"github.com/convox/rack/pkg/structs"
	"github.com/pkg/errors"

	cv "github.com/convox/version"
)

const (
	aesKey   = "AES256Key-32Characters1234567890"
	nonceHex = "37b8e8a308c354048d245f6d"
)

var (
	launcher = template.Must(template.New("launcher").Parse(launcherTemplate()))
)

func (p *Provider) SystemDecrypt(data []byte) ([]byte, error) {
	log := p.logger("SystemDecrypt")

	block, err := aes.NewCipher([]byte(aesKey))
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	dec, err := aesgcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	return dec, log.Success()
}

func (p *Provider) SystemEncrypt(data []byte) ([]byte, error) {
	log := p.logger("SystemEncrypt")

	block, err := aes.NewCipher([]byte(aesKey))
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	enc, err := aesgcm.Seal(nil, nonce, data, nil), nil
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	return enc, log.Success()
}

func (p *Provider) SystemGet() (*structs.System, error) {
	log := p.logger("SystemGet")

	system := &structs.System{
		Domain:   fmt.Sprintf("rack.%s", p.Rack),
		Name:     p.Rack,
		Provider: "local",
		Region:   "local",
		Status:   "running",
		Version:  p.Version,
	}

	return system, log.Success()
}

func (p *Provider) SystemInstall(w io.Writer, opts structs.SystemInstallOptions) (string, error) {
	name := cs(opts.Name, "convox")

	var version string

	if opts.Version != nil {
		version = *opts.Version
	} else {
		v, err := cv.Latest()
		if err != nil {
			return "", err
		}
		version = v
	}

	id := cs(opts.Id, "")

	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	u, err := user.Current()
	if err != nil {
		return "", err
	}

	if u.Uid != "0" {
		return "", fmt.Errorf("must be root to install a local rack")
	}

	fmt.Fprintf(w, "pulling: convox/rack:%s\n", version)

	if err := launcherInstall("router", w, opts, exe, "router"); err != nil {
		return "", err
	}

	if err := launcherInstall(fmt.Sprintf("rack.%s", name), w, opts, exe, "rack", "start", "--id", id, "--name", name); err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://rack.%s", name)

	fmt.Fprintf(w, "waiting for rack... ")

	tick := time.Tick(2 * time.Second)
	timeout := time.After(30 * time.Minute)

	ht := *(http.DefaultTransport.(*http.Transport))
	ht.TLSClientConfig.InsecureSkipVerify = true
	hc := &http.Client{Transport: &ht}

	for {
		select {
		case <-tick:
			_, err := hc.Get(url)
			if err == nil {
				fmt.Fprintf(w, "OK\n")
				return url, nil
			}
		case <-timeout:
			return "", fmt.Errorf("timeout")
		}
	}
}

func (p *Provider) SystemLogs(opts structs.LogsOptions) (io.ReadCloser, error) {
	log := p.logger("SystemLogs")

	r, w := io.Pipe()

	hostname, err := os.Hostname()
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	args := []string{"logs"}

	if opts.Follow == nil || *opts.Follow {
		args = append(args, "-f")
	}

	if opts.Since != nil {
		args = append(args, "--since", time.Now().UTC().Add((*opts.Since)*-1).Format(time.RFC3339))
	}

	args = append(args, hostname)

	cmd := exec.Command("docker", args...)

	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Start(); err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	go func() {
		defer w.Close()
		cmd.Wait()
	}()

	return r, log.Success()
}

func (p *Provider) SystemMetrics(opts structs.MetricsOptions) (structs.Metrics, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemOptions() (map[string]string, error) {
	log := p.logger("SystemOptions")

	options := map[string]string{
		"streaming": "websocket",
	}

	return options, log.Success()
}

func (p *Provider) SystemProcesses(opts structs.SystemProcessesOptions) (structs.Processes, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemProxy(host string, port int, in io.Reader) (io.ReadCloser, error) {
	log := p.logger("SystemProxy").Append("host=%s port=%d", host, port)

	cn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	go io.Copy(cn, in)

	return cn, log.Success()
}

func (p *Provider) SystemReleases() (structs.Releases, error) {
	return nil, fmt.Errorf("unimplemented")
}

func (p *Provider) SystemUninstall(name string, w io.Writer, opts structs.SystemUninstallOptions) error {
	u, err := user.Current()
	if err != nil {
		return err
	}

	if u.Uid != "0" {
		return fmt.Errorf("must be root to uninstall a local rack")
	}

	launcherRemove("rack", w)
	launcherRemove(fmt.Sprintf("rack.%s", name), w)

	return nil
}

func (p *Provider) SystemUpdate(opts structs.SystemUpdateOptions) error {
	log := p.logger("SystemUpdate").Append("version=%q", *opts.Version)

	if opts.Version != nil {
		v := *opts.Version

		if err := ioutil.WriteFile("/var/convox/version", []byte(v), 0644); err != nil {
			return errors.WithStack(log.Error(err))
		}

		if err := exec.Command("docker", "pull", fmt.Sprintf("convox/rack:%s", v)).Run(); err != nil {
			return errors.WithStack(log.Error(err))
		}

		go func() {
			time.Sleep(1 * time.Second)
			os.Exit(0)
		}()
	}

	return log.Success()
}

func launcherInstall(name string, w io.Writer, opts structs.SystemInstallOptions, command string, args ...string) error {
	var buf bytes.Buffer

	params := map[string]interface{}{
		"Name":    name,
		"Command": command,
		"Args":    args,
		"Logs":    fmt.Sprintf("/var/log/convox/%s.log", name),
	}

	if err := launcher.Execute(&buf, params); err != nil {
		return err
	}

	path := launcherPath(name)

	fmt.Fprintf(w, "installing: %s\n", path)

	if err := ioutil.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return err
	}

	if err := launcherStart(name); err != nil {
		return err
	}

	return nil
}

func launcherRemove(name string, w io.Writer) error {
	path := launcherPath(name)

	fmt.Fprintf(w, "removing: %s\n", path)

	launcherStop(name)

	os.Remove(path)

	return nil
}
