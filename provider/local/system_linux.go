package local

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"

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
	switch {
	case helpers.FileExists("/etc/systemd/resolved.conf"):
		return dnsInstallResolved(name)
	case helpers.FileExists("/etc/NetworkManager/NetworkManager.conf"):
		return dnsInstallNetworkManager(name)
	default:
		return fmt.Errorf("unable to install dns handlers")
	}
}

func dnsInstallNetworkManager(name string) error {
	data := []byte("[main]\ndns=dnsmasq\n")

	if err := helpers.WriteFile("/etc/NetworkManager/conf.d/convox.conf", data, 0644); err != nil {
		return err
	}

	rip, err := routerIP()
	if err != nil {
		return err
	}

	data = []byte(fmt.Sprintf("server=/%s/%s\n", name, rip))

	if err := helpers.WriteFile(fmt.Sprintf("/etc/NetworkManager/dnsmasq.d/%s", name), data, 0644); err != nil {
		return err
	}

	if err := exec.Command("systemctl", "restart", "NetworkManager").Run(); err != nil {
		return err
	}

	return nil
}

func dnsInstallResolved(name string) error {
	rip, err := routerIP()
	if err != nil {
		return err
	}

	data := []byte(fmt.Sprintf("[Resolve]\nDNS=%s\nDomains=~%s", rip, name))

	if err := helpers.WriteFile(fmt.Sprintf("/usr/lib/systemd/resolved.conf.d/convox.%s.conf", name), data, 0644); err != nil {
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

func dnsPort() string {
	return "53"
}

func dnsUninstall(name string) error {
	os.Remove(fmt.Sprintf("/etc/NetworkManager/dnsmasq.d/%s", name))
	os.Remove(fmt.Sprintf("/usr/lib/systemd/resolved.conf.d/convox.%s.conf", name))
	exec.Command("systemctl", "daemon-reload").Run()
	exec.Command("systemctl", "restart", "NetworkManager").Run()
	exec.Command("systemctl", "restart", "systemd-networkd", "systemd-resolved").Run()

	return nil
}

func networkSetup() error {
	switch {
	case helpers.FileExists("/etc/network/if-up.d"):
		return networkSetupDebian()
	case helpers.FileExists("/etc/NetworkManager/dispatcher.d"):
		return networkSetupRedhat()
	default:
		return fmt.Errorf("unable to set up network")
	}
}

func networkSetupDebian() error {
	data := []byte("#!/bin/sh\niptables -P FORWARD ACCEPT\n")

	if err := helpers.WriteFile("/etc/network/if-up.d/convox", data, 0755); err != nil {
		return fmt.Errorf("unable to set up network")
	}

	return nil
}

func networkSetupRedhat() error {
	data := []byte("#!/bin/sh\niptables -P FORWARD ACCEPT\n")

	if err := helpers.WriteFile("/etc/NetworkManager/dispatcher.d/99-convox", data, 0755); err != nil {
		return fmt.Errorf("unable to set up network")
	}

	return nil
}

func removeOriginalRack(name string) error {
	exec.Command("systemctl", "stop", fmt.Sprintf("convox.%s", name))
	os.Remove(fmt.Sprintf("/lib/systemd/system/convox.%s.service", name))

	return nil
}

func routerIP() (string, error) {
	data, err := exec.Command("kubectl", "get", "service/resolver", "-n", "convox-system", "--template={{.spec.clusterIP}}").CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func trustCertificate(name string, data []byte) error {
	switch {
	case helpers.FileExists("/usr/local/share/ca-certificates"):
		return trustCertificateDebian(name, data)
	case helpers.FileExists("/etc/pki/ca-trust/source/anchors"):
		return trustCertificateRedhat(name, data)
	default:
		return fmt.Errorf("unable to add ca certificate to trusted roots")
	}
}

func trustCertificateDebian(name string, data []byte) error {
	if err := helpers.WriteFile(fmt.Sprintf("/usr/local/share/ca-certificates/convox.%s.crt", name), data, 0644); err != nil {
		return fmt.Errorf("unable to add ca certificate to trusted roots")
	}

	if err := exec.Command("update-ca-certificates").Run(); err != nil {
		return fmt.Errorf("unable to add ca certificate to trusted roots")
	}

	return nil
}

func trustCertificateRedhat(name string, data []byte) error {
	if err := helpers.WriteFile(fmt.Sprintf("/etc/pks/ca-trust/source/anchors/convox.%s.crt", name), data, 0644); err != nil {
		return fmt.Errorf("unable to add ca certificate to trusted roots, try installing the 'ca-certificates' package")
	}

	if err := exec.Command("update-ca-trust", "force-enable").Run(); err != nil {
		return fmt.Errorf("unable to add ca certificate to trusted roots, try installing the 'ca-certificates' package")
	}

	if err := exec.Command("update-ca-trust", "extract").Run(); err != nil {
		return fmt.Errorf("unable to add ca certificate to trusted roots, try installing the 'ca-certificates' package")
	}

	return nil
}
