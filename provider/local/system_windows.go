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
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}

	crt := filepath.Join(tmp, "ca.crt")

	defer os.Remove(crt)

	if err := ioutil.WriteFile(crt, data, 0600); err != nil {
		return err
	}

	out, err := powershell(fmt.Sprintf(`Import-Certificate -CertStoreLocation Cert:\LocalMachine\Root -FilePath %s`, crt))
	fmt.Printf("string(out) = %+v\n", string(out))
	fmt.Printf("err = %+v\n", err)
	if err != nil {
		return err
	}

	return nil
}
