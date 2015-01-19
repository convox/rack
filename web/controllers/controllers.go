package controllers

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

var Templates = make(map[string]*template.Template)

func displayHelpers() template.FuncMap {
	return template.FuncMap{
		"join": func(s []string, t string) string {
			return strings.Join(s, t)
		},
		"meter": func(klass string, value int, total int) template.HTML {
			return template.HTML(fmt.Sprintf(`<div class="meter %s"><span style="width: %0.2f%%"></div>`, klass, float64(value)/float64(total)*100))
		},
	}
}

func RegisterTemplate(name string, names ...string) {
	templates := []string{}
	for _, name := range names {
		templates = append(templates, fmt.Sprintf("templates/%s.tmpl", name))
	}
	Templates[name] = template.Must(template.New("layout").Funcs(displayHelpers()).ParseFiles(templates...))
}

func RenderTemplate(rw http.ResponseWriter, name string, data interface{}) error {
	if _, ok := Templates[name]; !ok {
		return RenderError(rw, fmt.Errorf("no such template: %s", name))
	}
	if err := Templates[name].Execute(rw, data); err != nil {
		return RenderError(rw, err)
	}
	return nil
}

func RenderError(rw http.ResponseWriter, err error) error {
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
	return err
}

func Redirect(rw http.ResponseWriter, r *http.Request, path string) {
	http.Redirect(rw, r, path, http.StatusFound)
}
