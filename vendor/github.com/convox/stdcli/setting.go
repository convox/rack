package stdcli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
)

func (e *Engine) LocalSetting(name string) string {
	file := filepath.Join(e.localSettingDir(), name)

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return ""
	}

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(data))
}

func (e *Engine) SettingDelete(name string) error {
	file, err := e.settingFile(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil
	}

	if err := os.Remove(file); err != nil {
		return err
	}

	return nil
}

func (e *Engine) SettingDirectory(name string) (string, error) {
	dir, err := e.settingFile(name)
	if err != nil {
		return "", err
	}

	return dir, nil
}

func (e *Engine) SettingRead(name string) (string, error) {
	file, err := e.settingFile(name)
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadFile(file)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func (e *Engine) SettingReadKey(name, key string) (string, error) {
	s, err := e.SettingRead(name)
	if err != nil {
		return "", err
	}

	data := []byte(coalesce(s, "{}"))

	var kv map[string]string

	if err := json.Unmarshal(data, &kv); err != nil {
		return "", err
	}

	return kv[key], nil
}

func (e *Engine) SettingWrite(name, value string) error {
	file, err := e.settingFile(name)
	if err != nil {
		return err
	}

	dir := filepath.Dir(file)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	if err := ioutil.WriteFile(file, []byte(value), 0600); err != nil {
		return err
	}

	return nil
}

func (e *Engine) SettingWriteKey(name, key, value string) error {
	s, err := e.SettingRead(name)
	if err != nil {
		return err
	}

	data := []byte(coalesce(s, "{}"))

	var kv map[string]string

	if err := json.Unmarshal(data, &kv); err != nil {
		return err
	}

	kv[key] = value

	data, err = json.MarshalIndent(kv, "", "  ")
	if err != nil {
		return err
	}

	return e.SettingWrite(name, string(data))
}

func (e *Engine) localSettingDir() string {
	return fmt.Sprintf(".%s", e.Name)
}

func (e *Engine) settingFile(name string) (string, error) {
	if dir := e.Settings; dir != "" {
		return filepath.Join(dir, name), nil
	}

	dir, err := xdg.ConfigFile(fmt.Sprintf("%s/%s", e.Name, name))
	if err != nil {
		return "", err
	}

	return dir, nil
}
