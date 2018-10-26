package stdapi

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/gobuffalo/packr"
	"github.com/pkg/errors"
)

var (
	templateBox     packr.Box
	templateHelpers TemplateHelpers
)

type TemplateHelpers func(c *Context) template.FuncMap

func LoadTemplates(box packr.Box, helpers TemplateHelpers) {
	templateBox = box
	templateHelpers = helpers
}

func RenderTemplate(c *Context, path string, params interface{}) error {
	files := []string{}

	files = append(files, "layout.tmpl")
	files = append(files, filepath.Join(filepath.Dir(path), "layout.tmpl"))
	files = append(files, fmt.Sprintf("%s.tmpl", path))

	ts := template.New("main")

	if templateHelpers != nil {
		ts = ts.Funcs(templateHelpers(c))
	}

	for _, f := range files {
		if templateBox.Has(f) {
			if _, err := ts.Parse(templateBox.String(f)); err != nil {
				return errors.WithStack(err)
			}
		}
	}

	var buf bytes.Buffer

	if err := ts.Execute(&buf, params); err != nil {
		return errors.WithStack(err)
	}

	if _, err := io.Copy(c, &buf); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func appendIfExists(files []string, path string) []string {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		files = append(files, path)
	}

	return files
}
