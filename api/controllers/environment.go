package controllers

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/convox/rack/api/models"
)

func EnvironmentList(rw http.ResponseWriter, r *http.Request) error {
	app := mux.Vars(r)["app"]

	env, err := models.GetEnvironment(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	if err != nil {
		return err
	}

	return RenderJson(rw, env)
}

func EnvironmentSet(rw http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)

	app := vars["app"]

	_, err := models.GetEnvironment(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		return err
	}

	err = models.PutEnvironment(app, models.LoadEnvironment(body))

	if err != nil {
		return err
	}

	env, err := models.GetEnvironment(app)

	if err != nil {
		return err
	}

	return RenderJson(rw, env)
}

// func EnvironmentCreate(rw http.ResponseWriter, r *http.Request) {
//   vars := mux.Vars(r)

//   app := vars["app"]
//   name := vars["name"]
//   value := GetForm(r, "value")

//   env, err := models.GetEnvironment(app)

//   if err != nil {
//     helpers.Error(nil, err)
//     RenderError(rw, err)
//     return
//   }

//   env[strings.ToUpper(name)] = value

//   err = models.PutEnvironment(app, env)

//   if err != nil {
//     helpers.Error(nil, err)
//     RenderError(rw, err)
//     return
//   }

//   RenderText(rw, "ok")
// }

func EnvironmentDelete(rw http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	app := vars["app"]
	name := vars["name"]

	env, err := models.GetEnvironment(app)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	if err != nil {
		return err
	}

	delete(env, name)

	err = models.PutEnvironment(app, env)

	if err != nil {
		return err
	}

	env, err = models.GetEnvironment(app)

	if err != nil {
		return err
	}

	return RenderJson(rw, env)
}
