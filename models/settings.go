package models

import "os"

func SettingGet(name string) (string, error) {
	value, err := s3Get(os.Getenv("SETTINGS_BUCKET"), name)

	if err != nil {
		return "", err
	}

	return string(value), nil
}

func SettingSet(name, value string) error {
	return s3Put(os.Getenv("SETTINGS_BUCKET"), name, []byte(value), false)
}
