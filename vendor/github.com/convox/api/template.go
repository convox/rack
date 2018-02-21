package api

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

var (
	Templates = map[string]*template.Template{}
)

func LoadTemplates(dir string, helpers map[string]interface{}) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			files := []string{}

			files = appendIfExists(files, filepath.Join(dir, "layout.tmpl"))
			files = appendIfExists(files, filepath.Join(filepath.Dir(path), "layout.tmpl"))
			files = append(files, path)

			t, err := template.New("main").Funcs(helpers).ParseFiles(files...)

			if err != nil {
				return err
			}

			rel, err := filepath.Rel(dir, path)

			if err != nil {
				return err
			}

			Templates[rel] = t
		}

		return nil
	})
}

func RenderTemplate(w http.ResponseWriter, path string, params interface{}) error {
	t, ok := Templates[path]

	if !ok {
		return fmt.Errorf("no such template: %s", path)
	}

	if err := t.Execute(w, params); err != nil {
		return err
	}

	return nil
}

func appendIfExists(files []string, path string) []string {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		files = append(files, path)
	}

	return files
}
