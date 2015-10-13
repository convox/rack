package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/convox/rack/api/httperr"
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

func RenderError(rw http.ResponseWriter, err error) *httperr.Error {
	rw.Write([]byte(fmt.Sprintf(`{"error":%q}`, err.Error())))

	return httperr.Server(err)
}

func RenderJson(rw http.ResponseWriter, object interface{}) *httperr.Error {
	data, err := json.MarshalIndent(object, "", "  ")

	if err != nil {
		return RenderError(rw, err)
	}

	data = append(data, '\n')

	rw.Header().Set("Content-Type", "application/json")

	_, err = rw.Write(data)

	return httperr.Server(err)
}

func RenderText(rw http.ResponseWriter, text string) *httperr.Error {
	_, err := rw.Write([]byte(text))

	return httperr.Server(err)
}

func RenderSuccess(rw http.ResponseWriter) *httperr.Error {
	_, err := rw.Write([]byte(`{"success":true}`))

	return httperr.Server(err)
}

func Redirect(rw http.ResponseWriter, r *http.Request, path string) *httperr.Error {
	http.Redirect(rw, r, path, http.StatusFound)

	return nil
}
