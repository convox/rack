package controllers

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

var Templates = make(map[string]*template.Template)

func displayHelpers() template.FuncMap {
	return template.FuncMap{
		"duration": func(start, end time.Time) string {
			if end.IsZero() {
				return "--"
			} else {
				duration := end.Sub(start)
				seconds := duration / time.Second
				return fmt.Sprintf("%dmin %dsec", seconds/60, seconds%60)
			}
		},
		"join": func(s []string, t string) string {
			return strings.Join(s, t)
		},
		"label": func(name, value string) template.HTML {
			return template.HTML(fmt.Sprintf(`<div class="labelled-value" id="%s"><span class="name">%s</span><span class="value">%s</span></div>`, strings.ToLower(name), name, value))
		},
		"meter": func(klass string, value int, total int) template.HTML {
			return template.HTML(fmt.Sprintf(`<div class="meter %s"><span style="width: %0.2f%%"></div>`, klass, float64(value)/float64(total)*100))
		},
		"status": func(s string) string {
			state := "default"
			switch s {
			case "running":
				state = "success"
			case "updating":
				state = "warning"
			}
			return fmt.Sprintf(`<div class="label label-%s">%s</div>`, state, s)
		},
		"timeago": func(t time.Time) template.HTML {
			return template.HTML(fmt.Sprintf(`<span class="timeago" title="%s">%s</span>`, t.Format(time.RFC3339), t.Format("2006-01-02 15:04:05 UTC")))
		},
	}
}

func ParseForm(r *http.Request) map[string]string {
	options := make(map[string]string)

	r.ParseMultipartForm(4096)

	for key, values := range r.PostForm {
		options[key] = values[0]
	}

	return options
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
