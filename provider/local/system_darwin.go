package local

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"

	"github.com/convox/rack/pkg/helpers"
)

func checkPermissions() error {
	u, err := user.Current()
	if err != nil {
		return err
	}

	if u.Uid != "0" {
		return fmt.Errorf("must be run as root")
	}

	return nil
}

func dnsInstall(name string) error {
	os.Remove(fmt.Sprintf("/etc/resolver/%s", name))

	if err := helpers.WriteFile(fmt.Sprintf("/etc/resolver/%s", name), []byte("nameserver 127.0.0.1\nport 5453\n"), 0644); err != nil {
		return fmt.Errorf("could not write resolver config")
	}

	exec.Command("killall", "-HUP", "mDNSResponder").Run()

	return nil
}

func dnsPort() string {
	return "5453"
}

func dnsUninstall(name string) error {
	os.Remove(fmt.Sprintf("/etc/resolver/%s", name))
	return nil
}

func dockerSocket() string {
	return "/var/run/docker.sock"
}

func networkSetup() error {
	return nil
}

func removeOriginalRack(name string) error {
	os.Remove(fmt.Sprintf("/Library/LaunchDaemons/convox.rack.%s.plist", name))
	os.Remove("/Library/LaunchDaemons/convox.router.plist")

	exec.Command("launchctl", "remove", fmt.Sprintf("convox.rack.%s", name)).Run()
	exec.Command("launchctl", "remove", "convox.rack").Run()
	exec.Command("launchctl", "remove", "convox.router").Run()

	return nil
}

func trustCertificate(name string, data []byte) error {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}

	crt := filepath.Join(tmp, "ca.crt")

	defer os.Remove(crt)

	if err := ioutil.WriteFile(crt, data, 0600); err != nil {
		return err
	}

	if err := exec.Command("security", "add-trusted-cert", "-d", "-r", "trustRoot", "-k", "/Library/Keychains/System.keychain", crt).Run(); err != nil {
		return fmt.Errorf("unable to add ca certificate to trusted roots")
	}

	if err := installDockerDesktopCertificate(data); err != nil {
		return fmt.Errorf("unable to install docker desktop ca: %v", err)
	}

	return nil
}
