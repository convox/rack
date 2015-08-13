package controllers

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/websocket"

	"github.com/convox/kernel/helpers"
	"github.com/convox/kernel/models"
)

func init() {
	RegisterPartial("service", "logs")
	RegisterPartial("services", "names")

	RegisterTemplate("service", "layout", "service")
	RegisterTemplate("services", "layout", "services")
	// RegisterTemplate("app", "layout", "app")
}

func ServiceList(rw http.ResponseWriter, r *http.Request) {
	log := servicesLogger("list").Start()

	services, err := models.ListServiceStacks()

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderJson(rw, services)
}

func ServiceShow(rw http.ResponseWriter, r *http.Request) {
	log := servicesLogger("show").Start()

	name := mux.Vars(r)["service"]

	service, err := models.GetServiceFromName(name)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	// sort.Sort(services)

	RenderTemplate(rw, "service", service)
}

func ServiceNameList(rw http.ResponseWriter, r *http.Request) {
	log := servicesLogger("nameList").Start()

	t := mux.Vars(r)["type"]

	services, err := models.ListServiceStacks()

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	s := models.Services{}

	for _, item := range services {
		if item.Tags["Service"] == t {
			s = append(s, item)
		}
	}

	RenderPartial(rw, "services", "names", s)
}

func ServiceCreate(rw http.ResponseWriter, r *http.Request) {
	log := servicesLogger("create").Start()

	name := GetForm(r, "name")
	t := GetForm(r, "type")

	password, err := rand_password(20)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	service := &models.Service{
		Name:     name,
		Password: password,
		Type:     t,
	}

	err = service.Create()

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	Redirect(rw, r, "/services")
}

func ServiceDelete(rw http.ResponseWriter, r *http.Request) {
	log := servicesLogger("delete").Start()

	vars := mux.Vars(r)
	name := vars["service"]

	service, err := models.GetServiceFromName(name)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	log.Success("step=services.get service=%q", service.Name)

	err = service.Delete()

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	log.Success("step=service.delete service=%q", service.Name)

	RenderText(rw, "ok")
}

func ServiceLink(rw http.ResponseWriter, r *http.Request) {
	log := servicesLogger("link").Start()

	vars := mux.Vars(r)

	app := vars["app"]
	name := GetForm(r, "name")
	stack := GetForm(r, "stack")

	err := models.LinkService(app, name, stack)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
	}

	Redirect(rw, r, fmt.Sprintf("/apps/%s", app))
}

func ServiceUnlink(rw http.ResponseWriter, r *http.Request) {
	log := servicesLogger("unlink").Start()

	vars := mux.Vars(r)

	app := vars["app"]
	name := vars["name"]

	err := models.UnlinkService(app, name)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
	}

	RenderText(rw, "ok")
}

func ServiceLogs(rw http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["service"]

	service, err := models.GetServiceFromName(name)

	if err != nil {
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "service", "logs", service)
}

func ServiceStream(rw http.ResponseWriter, r *http.Request) {
	log := servicesLogger("stream").Start()

	service, err := models.GetServiceFromName(mux.Vars(r)["service"])

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	logs := make(chan []byte)
	done := make(chan bool)

	service.SubscribeLogs(logs, done)

	ws, err := upgrader.Upgrade(rw, r, nil)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	log.Success("step=upgrade service=%q", service.Name)

	defer ws.Close()

	for data := range logs {
		ws.WriteMessage(websocket.TextMessage, data)
	}

	log.Success("step=ended service=%q", service.Name)
}

func rand_password(length int) (string, error) {
	// Take from https://github.com/cmiceli/password-generator-go

	var chars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$%^&*()-_=+,.?:;{}[]`~") // no /@" and space allowed for RDS password

	new_pword := make([]byte, length)
	random_data := make([]byte, length+(length/4)) // storage for random bytes.
	clen := byte(len(chars))
	maxrb := byte(256 - (256 % len(chars)))
	i := 0
	for {
		if _, err := io.ReadFull(rand.Reader, random_data); err != nil {
			return "", err
		}
		for _, c := range random_data {
			if c >= maxrb {
				continue
			}
			new_pword[i] = chars[c%clen]
			i++
			if i == length {
				return string(new_pword), nil
			}
		}
	}
	return "", fmt.Errorf("unreachable")
}

func servicesLogger(at string) *logger.Logger {
	return logger.New("ns=kernel cn=services").At(at)
}
