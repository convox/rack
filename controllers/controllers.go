package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws/awserr"
)

func awsError(err error) string {
	if ae, ok := err.(awserr.Error); ok {
		return ae.Code()
	}

	return ""
}

func GetForm(r *http.Request, name string) string {
	r.ParseMultipartForm(4096)

	if len(r.PostForm[name]) == 1 {
		return r.PostForm[name][0]
	} else {
		return ""
	}
}

func RenderError(rw http.ResponseWriter, err error) error {
	if err != nil {
		http.Error(rw, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
	}

	return err
}

func RenderJson(rw http.ResponseWriter, object interface{}) error {
	data, err := json.Marshal(object)

	if err != nil {
		return RenderError(rw, err)
	}

	rw.Header().Set("Content-Type", "application/json")

	_, err = rw.Write(data)

	return err
}

func RenderText(rw http.ResponseWriter, text string) error {
	_, err := rw.Write([]byte(text))
	return err
}

func RenderSuccess(rw http.ResponseWriter) error {
	_, err := rw.Write([]byte(`{"success":true}`))

	return err
}

func RenderForbidden(rw http.ResponseWriter, message string) error {
	rw.WriteHeader(403)

	_, err := rw.Write([]byte(fmt.Sprintf(`{"error":%q}`, message)))

	return err
}

func RenderNotFound(rw http.ResponseWriter, message string) error {
	rw.WriteHeader(404)

	_, err := rw.Write([]byte(fmt.Sprintf(`{"error":%q}`, message)))

	return err
}

func Redirect(rw http.ResponseWriter, r *http.Request, path string) error {
	http.Redirect(rw, r, path, http.StatusFound)

	return nil
}
