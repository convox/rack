package router

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func execute(command string, args ...string) error {
	cmd := exec.Command(command, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func incrementIP(ip net.IP, num uint32) net.IP {
	in := make(net.IP, len(ip))
	binary.BigEndian.PutUint32(in, binary.BigEndian.Uint32(ip)+num)
	return in
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}

func linuxRelease() (string, error) {
	os_info := exec.Command("/bin/cat", "/etc/os-release")
	var buf bytes.Buffer
	os_info.Stdout = &buf

	err := os_info.Run()

	if err != nil {
		return "unknown", err
	}

	output := buf.String()
	lines := strings.Split(output, "\n")
	os_identifier := make([]string, 2)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		parts := strings.Split(line, "=")
		key := parts[0]
		value := parts[1]
		value = strings.Replace(value, "\"", "", -1)
		value = strings.ToLower(value)
		if key == "NAME" {
			os_identifier[0] = value
		}
		if key == "VERSION_ID" {
			os_identifier[1] = value
		}
	}
	return strings.Join(os_identifier, "-"), nil
}
