package stdcli

import (
	"path/filepath"
	"strings"
)

func Default(value, def string) string {
	if value == "" {
		return def
	} else {
		return value
	}
}

func DefaultApp(config string) (string, error) {
	if app := ReadSetting("app"); app != "" {
		return app, nil
	}

	abs, err := filepath.Abs(filepath.Dir(config))
	if err != nil {
		return "", err
	}

	app := filepath.Base(abs)
	app = strings.ToLower(app)
	app = strings.Replace(app, ".", "-", -1)

	return app, nil
}
