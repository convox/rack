package stdapi

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

var (
	templates       FileSystem
	templateHelpers TemplateHelpers
)

type TemplateHelpers func(c *Context) template.FuncMap

type FileSystem http.FileSystem

func LoadTemplates(files FileSystem, helpers TemplateHelpers) {
	templates = files
	templateHelpers = helpers
}

func TemplateExists(path string) bool {
	_, err := templates.Open(path)
	return !os.IsNotExist(err)
}

func RenderTemplate(c *Context, path string, params interface{}) error {
	return RenderTemplatePart(c, path, "main", params)
}

func RenderTemplatePart(c *Context, path, part string, params interface{}) error {
	files := []string{}

	files = append(files, "layout.tmpl")

	parts := strings.Split(filepath.Dir(path), "/")

	for i := range parts {
		files = append(files, filepath.Join(filepath.Join(parts[0:i+1]...), "layout.tmpl"))
	}

	files = append(files, fmt.Sprintf("%s.tmpl", path))

	ts := template.New(part)

	if templateHelpers != nil {
		ts = ts.Funcs(templateHelpers(c))
	}

	for _, f := range files {
		fd, err := templates.Open(f)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}

		data, err := ioutil.ReadAll(fd)
		if err != nil {
			return err
		}

		if _, err := ts.Parse(string(data)); err != nil {
			return errors.WithStack(err)
		}
	}

	var buf bytes.Buffer

	if err := ts.Execute(&buf, params); err != nil {
		return errors.WithStack(err)
	}

	io.Copy(c, &buf)

	return nil
}
