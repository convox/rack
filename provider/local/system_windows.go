package local

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func checkPermissions() error {
	data, err := powershell(`([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")`)
	if err != nil || strings.TrimSpace(string(data)) != "True" {
		return fmt.Errorf("must be run as administrator")
	}

	return nil
}

func dnsInstall(name string) error {
	if err := dnsUninstall(name); err != nil {
		return err
	}

	if _, err := powershell(fmt.Sprintf(`Add-DnsClientNrptRule -Namespace ".%s" -NameServers "127.0.0.1"`, name)); err != nil {
		return fmt.Errorf("unable to install dns handlers")
	}

	return nil
}

func dnsPort() string {
	return "53"
}

func dnsUninstall(name string) error {
	if _, err := powershell(fmt.Sprintf(`Get-DnsClientNrptRule | ForEach-Object -Process { if ($_.Namespace -eq ".%s") { Remove-DnsClientNrptRule -Force $_.Name } }`, name)); err != nil {
		return fmt.Errorf("unable to clear dns handlers")
	}

	return nil
}

func powershell(command string) ([]byte, error) {
	return exec.Command("powershell.exe", "-Sta", "-NonInteractive", "-ExecutionPolicy", "RemoteSigned", command).CombinedOutput()
}

func removeOriginalRack(name string) error {
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

	if _, err := powershell(fmt.Sprintf(`Import-Certificate -CertStoreLocation Cert:\LocalMachine\Root -FilePath %s`, crt)); err != nil {
		return fmt.Errorf("unable to add ca certificate to trusted roots")
	}

	return nil
}
