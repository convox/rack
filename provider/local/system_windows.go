package local

import (
	"fmt"
	"os/exec"
)

func checkPermissions() error {
	return nil
}

func dnsInstall(name string) error {
	data, err := powershell(fmt.Sprintf(`Add-DnsClientNrptRule -Namespace ".%s" -NameServers "127.0.0.1"`, name))
	fmt.Printf("string(data) = %+v\n", string(data))
	fmt.Printf("err = %+v\n", err)
	if err != nil {
		return err
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
	return exec.Command("powershell.exe", "-Sta", "-NonInteractive", "-ExecutionPolicy", "RemoteSigned", "-EncodedCommand", command).CombinedOutput()
}

func removeOriginalRack(name string) error {
	return nil
}

func trustCertificate(data []byte) error {
	return nil
}
