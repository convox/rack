package stdapi

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

var (
	templateDir     string
	templateHelpers TemplateHelpers
)

type TemplateHelpers func(c *Context) template.FuncMap

func LoadTemplates(dir string, helpers TemplateHelpers) {
	templateDir = dir
	templateHelpers = helpers
}

func RenderTemplate(c *Context, path string, params interface{}) error {
	files := []string{}

	files = appendIfExists(files, filepath.Join(templateDir, "layout.tmpl"))
	files = appendIfExists(files, filepath.Join(templateDir, filepath.Dir(path), "layout.tmpl"))
	files = append(files, filepath.Join(templateDir, fmt.Sprintf("%s.tmpl", path)))

	ts := template.New("main")

	if templateHelpers != nil {
		ts = ts.Funcs(templateHelpers(c))
	}

	t, err := ts.ParseFiles(files...)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	if err := t.Execute(&buf, params); err != nil {
		return errors.WithStack(err)
	}

	if _, err := io.Copy(c, &buf); err != nil {
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
