package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws/awserr"
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

func RenderError(rw http.ResponseWriter, err error) *HttpError {
	if err != nil {
		http.Error(rw, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
	}

	return ServerError(err)
}

func RenderJson(rw http.ResponseWriter, object interface{}) *HttpError {
	data, err := json.MarshalIndent(object, "", "  ")

	if err != nil {
		return RenderError(rw, err)
	}

	data = append(data, '\n')

	rw.Header().Set("Content-Type", "application/json")

	_, err = rw.Write(data)

	return ServerError(err)
}

func RenderText(rw http.ResponseWriter, text string) *HttpError {
	_, err := rw.Write([]byte(text))

	return ServerError(err)
}

func RenderSuccess(rw http.ResponseWriter) *HttpError {
	_, err := rw.Write([]byte(`{"success":true}`))

	return ServerError(err)
}

func Redirect(rw http.ResponseWriter, r *http.Request, path string) *HttpError {
	http.Redirect(rw, r, path, http.StatusFound)

	return nil
}
