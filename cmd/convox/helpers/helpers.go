package helpers

import (
	"os"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
)

func Exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

func HumanizeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	} else {
		return humanize.Time(t)
	}
}

func DetectApplication(dir string) string {
	switch {
	case Exists(filepath.Join(dir, "Procfile")):
		return "heroku"
	case Exists(filepath.Join(dir, "manage.py")):
		return "django"
	case Exists(filepath.Join(dir, "config/application.rb")):
		return "rails"
	case Exists(filepath.Join(dir, "config.ru")):
		return "sinatra"
	case Exists(filepath.Join(dir, "Gemfile.lock")):
		return "ruby"
	case Exists(filepath.Join(dir, "requirements.txt")):
		return "python"
	}

	return "unknown"
}
