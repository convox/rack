package helpers

import (
	"os"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
)

// Exists checks if a file exists
func Exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}

	return true
}

// HumanizeTime converts a Time into a human-friendly format
func HumanizeTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return humanize.Time(t)
}

// DetectApplication detects an apps type by looking for special files
func DetectApplication(dir string) string {
	switch {
	case Exists(filepath.Join(dir, "Procfile")):
		switch {
		case Exists(filepath.Join(dir, "requirements.txt")) || Exists(filepath.Join(dir, "setup.py")):
			return "heroku/python"
		case Exists(filepath.Join(dir, "package.json")):
			return "heroku/nodejs"
		case Exists(filepath.Join(dir, "Gemfile")):
			return "heroku/ruby"
		}

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
