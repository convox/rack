package structs

import "encoding/json"

type Settings map[string]string

func LoadSettings(data []byte) (Settings, error) {
	var settings Settings

	err := json.Unmarshal(data, &settings)

	if err != nil {
		return nil, err
	}

	return settings, nil
}
