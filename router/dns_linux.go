package router

import (
	"fmt"
	"net"
)

func (d *DNS) setupResolver(domain string, ip net.IP) error {
	version, err := linuxRelease()
	switch version {
	case "ubuntu-18.04":
		if err := installUbuntu1804(domain, ip); err != nil {
			return err
		}
	default:
		if err := installGenericLinux(domain, ip); err != nil {
			return err
		}
	}
	return nil
}

func installUbuntu1804(domain string, ip net.IP) error {

	data := []byte(fmt.Sprintf("[Resolve]\nDNS=%s\nDomains=~%s", ip, domain))

	if err := writeFile("/usr/lib/systemd/resolved.conf.d/convox.conf", data); err != nil {
		return err
	}

	if err := execute("systemctl", "daemon-reload"); err != nil {
		return err
	}

	if err := execute("systemctl", "restart", "systemd-networkd"); err != nil {
		return err
	}

	if err := execute("systemctl", "restart", "systemd-resolved"); err != nil {
		return err
	}
	return nil
}

func installGenericLinux(domain string, ip net.IP) error {

	data := []byte("[main]\ndns=dnsmasq\n")

	if err := writeFile("/etc/NetworkManager/conf.d/convox.conf", data); err != nil {
		return err
	}

	data = []byte(fmt.Sprintf("server=/%s/%s\n", domain, ip))

	if err := writeFile(fmt.Sprintf("/etc/NetworkManager/dnsmasq.d/%s", domain), data); err != nil {
		return err
	}

	if err := execute("systemctl", "restart", "NetworkManager"); err != nil {
		return err
	}
	return nil
}
