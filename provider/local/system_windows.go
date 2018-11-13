package local

import (
	"fmt"
	"os/exec"
	"strings"
)

func checkPermissions() error {
	data, err := powershell(`([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")`)
	if err != nil || strings.TrimSpace(string(data)) != "True" {
		return fmt.Errorf("must be run in powershell running as administrator")
	}

	return nil
}

func dnsInstall(name string) error {
	if _, err := powershell(fmt.Sprintf(`Add-DnsClientNrptRule -Namespace ".%s" -NameServers "127.0.0.1"`, name)); err != nil {
		return fmt.Errorf("unable to install dns handlers")
	}

	return nil
}

func dnsPort() string {
	return "53"
}

func dnsUninstall(name string) error {
	return nil
}

func powershell(command string) ([]byte, error) {
	return exec.Command("powershell.exe", "-Sta", "-NonInteractive", "-ExecutionPolicy", "RemoteSigned", command).CombinedOutput()
}

func removeOriginalRack(name string) error {
	return nil
}

func trustCertificate(data []byte) error {
	return nil
}
