package local

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/convox/rack/pkg/helpers"
)

func dnsInstall(name string) error {
	v, err := helpers.LinuxRelease()
	if err != nil {
		return err
	}

	switch v {
	case "ubuntu-18.04":
		return dnsInstallLinuxResolved(name)
	default:
		return dnsInstallLinuxNetworkManager(name)
	}
}

func dnsInstallLinuxNetworkManager(name string) error {
	data := []byte("[main]\ndns=dnsmasq\n")

	if err := ioutil.WriteFile("/etc/NetworkManager/conf.d/convox.conf", data, 0644); err != nil {
		return err
	}

	data = []byte(fmt.Sprintf("server=/%s/127.0.0.1:5453\n", name))

	if err := ioutil.WriteFile(fmt.Sprintf("/etc/NetworkManager/dnsmasq.d/%s", name), data, 0644); err != nil {
		return err
	}

	if err := exec.Command("systemctl", "restart", "NetworkManager").Run(); err != nil {
		return err
	}

	return nil
}

func dnsInstallLinuxResolved(name string) error {
	data := []byte(fmt.Sprintf("[Resolve]\nDNS=127.0.0.1:5453\nDomains=~%s", name))

	if err := ioutil.WriteFile(fmt.Sprintf("/usr/lib/systemd/resolved.conf.d/convox.%s.conf", name), data, 0644); err != nil {
		return err
	}

	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return err
	}

	if err := exec.Command("systemctl", "restart", "systemd-networkd", "systemd-resolved").Run(); err != nil {
		return err
	}

	return nil
}

func dnsUninstall(name string) error {
	os.Remove(fmt.Sprintf("/etc/NetworkManager/dnsmasq.d/%s", name))
	os.Remove(fmt.Sprintf("/usr/lib/systemd/resolved.conf.d/convox.%s.conf", name))
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "restart", "NetworkManager").Run()
	exec.Command("systemctl", "restart", "systemd-networkd", "systemd-resolved").Run()

	return nil
}

func removeOriginalRack(name string) error {
	exec.Command("systemctl", "stop", fmt.Sprintf("convox.%s", name))
	os.Remove(fmt.Sprintf("/lib/systemd/system/convox.%s.service", name))

	return nil
}

func trustCertificate(data []byte) error {
	return nil
}
