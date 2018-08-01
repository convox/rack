package router

import (
	"encoding/binary"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
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
