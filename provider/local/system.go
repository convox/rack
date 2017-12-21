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
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"text/template"
	"time"

	"github.com/convox/rack/structs"
	"github.com/kr/text"
	"github.com/pkg/errors"
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
		Image:   fmt.Sprintf("convox/rack:%s", p.Version),
		Name:    p.Name,
		Status:  "running",
		Version: p.Version,
	}

	return system, log.Success()
}

func (p *Provider) SystemInstall(name string, opts structs.SystemInstallOptions) (string, error) {
	cx, err := os.Executable()
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

	if opts.Output != nil {
		fmt.Fprintf(opts.Output, "pulling: convox/rack:%s\n", opts.Version)
	}

	if opts.Version == nil {
		return "", fmt.Errorf("must specify a version")
	}

	vf := "/var/convox/version"

	switch runtime.GOOS {
	case "darwin":
		vf = "/Users/Shared/convox/version"
	}

	if err := ioutil.WriteFile(vf, []byte(*opts.Version), 0644); err != nil {
		return "", err
	}

	cmd := exec.Command("docker", "pull", fmt.Sprintf("convox/rack:%s", *opts.Version))

	if opts.Output != nil {
		cmd.Stdout = text.NewIndentWriter(opts.Output, []byte("  "))
		cmd.Stderr = text.NewIndentWriter(opts.Output, []byte("  "))
	}

	if err := cmd.Run(); err != nil {
		return "", err
	}

	if err := launcherInstall("convox.router", opts, cx, "router"); err != nil {
		return "", err
	}

	if err := launcherInstall("convox.rack", opts, cx, "rack", "start"); err != nil {
		return "", err
	}

	return "https://localhost:5443", nil
}

func (p *Provider) SystemLogs(opts structs.LogsOptions) (io.ReadCloser, error) {
	log := p.logger("SystemLogs")

	r, w := io.Pipe()

	hostname, err := os.Hostname()
	if err != nil {
		return nil, errors.WithStack(log.Error(err))
	}

	args := []string{"logs"}

	if opts.Follow {
		args = append(args, "-f")
	}

	if !opts.Since.IsZero() {
		args = append(args, "--since", opts.Since.Format(time.RFC3339))
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

func (p *Provider) SystemUninstall(name string, opts structs.SystemInstallOptions) error {
	launcherRemove("convox.frontend")
	launcherRemove("convox.rack")
	launcherRemove("convox.router")

	exec.Command("launchctl", "remove", "convox.frontend").Run()
	exec.Command("launchctl", "remove", "convox.rack").Run()
	exec.Command("launchctl", "remove", "convox.router").Run()

	return nil
}

func (p *Provider) SystemUpdate(opts structs.SystemUpdateOptions) error {
	log := p.logger("SystemUpdate").Append("version=%q", opts.Version)

	w := opts.Output
	if w == nil {
		w = ioutil.Discard
	}

	if opts.Version != nil {
		v := *opts.Version

		w.Write([]byte("Restarting... OK\n"))

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

func launcherInstall(name string, opts structs.SystemInstallOptions, command string, args ...string) error {
	var buf bytes.Buffer

	params := map[string]interface{}{
		"Name":    name,
		"Command": command,
		"Args":    args,
		"Logs":    fmt.Sprintf("/var/log/%s.log", name),
	}

	if err := launcher.Execute(&buf, params); err != nil {
		return err
	}

	path := launcherPath(name)

	if opts.Output != nil {
		fmt.Fprintf(opts.Output, "installing: %s\n", path)
	}

	if err := ioutil.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return err
	}

	if err := launcherStart(name); err != nil {
		return err
	}

	return nil
}

func launcherRemove(name string) error {
	path := launcherPath(name)

	fmt.Printf("removing: %s\n", path)

	launcherStop(name)

	os.Remove(path)

	return nil
}
