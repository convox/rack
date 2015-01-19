package helpers

import "net/http"

func ParseForm(r *http.Request) map[string]string {
	options := make(map[string]string)

	r.ParseMultipartForm(4096)

	for key, values := range r.PostForm {
		options[key] = values[0]
	}

	return options
}
