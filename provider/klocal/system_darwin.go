package klocal

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

func dnsInstall(name string) error {
	os.Remove(fmt.Sprintf("/etc/resolver/%s", name))

	if err := ioutil.WriteFile(fmt.Sprintf("/etc/resolver/%s", name), []byte("nameserver 127.0.0.1\nport 5453\n"), 0644); err != nil {
		return fmt.Errorf("could not write resolver config")
	}

	return nil
}

func dnsUninstall(name string) error {
	os.Remove(fmt.Sprintf("/etc/resolver/%s", name))
	return nil
}

func removeOriginalRack(name string) error {
	exec.Command("launchctl", "remove", fmt.Sprintf("convox.rack.%s", name)).Run()
	os.Remove(fmt.Sprintf("/Library/LaunchDaemons/convox.rack.%s.plis", name))

	return nil
}

func trustCertificate(data []byte) error {
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

	return nil
}
