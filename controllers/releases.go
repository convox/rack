package controllers

import (
  "net/http"

  "github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
  "github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"

  "github.com/convox/kernel/helpers"
  "github.com/convox/kernel/models"
)

func ReleaseCreate(rw http.ResponseWriter, r *http.Request) {
  // app name, docker image tag and id
  log := releasesLogger("create").Start()

  vars := mux.Vars(r)
  name := vars["app"]

  manifest := GetForm(r, "manifest")
  tag := GetForm(r, "tag")

  app, err := models.GetApp(name)

  if err != nil {
    helpers.Error(log, err)
    RenderError(rw, err)
    return
  }

  release, err := app.ForkRelease()

  if err != nil {
    helpers.Error(log, err)
    RenderError(rw, err)
    return
  }

  build := models.NewBuild(app.Name)
  build.Id = tag
  build.Release = release.Id
  build.Status = "complete"

  release.Build = build.Id
  release.Manifest = manifest

  err = build.Save()

  if err != nil {
    helpers.Error(log, err)
    RenderError(rw, err)
    return
  }

  err = release.Save()

  if err != nil {
    helpers.Error(log, err)
    RenderError(rw, err)
    return
  }

  err = release.Promote()

  if err != nil {
    helpers.Error(log, err)
    RenderError(rw, err)
    return
  }

  log.Success("step=release.create app=%q", app.Name)

  RenderText(rw, "ok")
}

func releasesLogger(at string) *logger.Logger {
  return logger.New("ns=kernel cn=releases").At(at)
}
