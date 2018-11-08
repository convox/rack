package helpers

import (
	"bytes"

	yaml "gopkg.in/yaml.v2"
)

func FormatYAML(data []byte) ([]byte, error) {
	parts := bytes.Split(data, []byte("---"))

	for i, part := range parts {
		var v interface{}

		if err := yaml.Unmarshal(part, &v); err != nil {
			return nil, err
		}

		data, err := yaml.Marshal(v)
		if err != nil {
			return nil, err
		}

		parts[i] = data
	}

	return bytes.Join(parts, []byte("---\n")), nil
}
