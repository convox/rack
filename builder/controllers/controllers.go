package controllers

import "net/http"

func ParseForm(r *http.Request) map[string]string {
	options := make(map[string]string)

	r.ParseMultipartForm(4096)

	for key, values := range r.PostForm {
		options[key] = values[0]
	}

	return options
}

func RenderError(rw http.ResponseWriter, err error) error {
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}
	return err
}

func RenderText(rw http.ResponseWriter, text string) {
	rw.Write([]byte(text))
}

func Redirect(rw http.ResponseWriter, r *http.Request, path string) {
	http.Redirect(rw, r, path, http.StatusFound)
}
