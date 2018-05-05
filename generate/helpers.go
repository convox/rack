package generate

import (
	"bytes"
	"os/exec"
)

func gofmt(data []byte) ([]byte, error) {
	cmd := exec.Command("goimports")
	cmd.Stdin = bytes.NewReader(data)

	data, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	data = bytes.Replace(data, []byte("{\n\n"), []byte("{\n"), -1)
	data = bytes.Replace(data, []byte("\n\n}"), []byte("\n}"), -1)
	data = bytes.Replace(data, []byte("\n\n\tr.Route"), []byte("\n\tr.Route"), -1)

	return data, nil
}
