package controllers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"

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
		log.Error(err)
		RenderError(rw, err)
		return
	}

	// sort.Sort(services)

	RenderTemplate(rw, "services", services)
}

func ServiceShow(rw http.ResponseWriter, r *http.Request) {
	log := servicesLogger("show").Start()

	name := mux.Vars(r)["service"]

	service, err := models.GetServiceFromName(name)

	if err != nil {
		log.Error(err)
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
		log.Error(err)
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
	password := GetForm(r, "password")
	t := GetForm(r, "type")

	service := &models.Service{
		Name:     name,
		Password: password,
		Type:     t,
	}

	err := service.Create()

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	Redirect(rw, r, "/services")
}

func ServiceLink(rw http.ResponseWriter, r *http.Request) {
	// log := servicesLogger("link").Start()

	app := mux.Vars(r)["app"]
	name := GetForm(r, "name")
	t := GetForm(r, "type")
	stack := GetForm(r, "stack")

	service := &models.Service{
		App:   app,
		Name:  name,
		Type:  t,
		Stack: stack,
	}

	s, err := models.GetServiceFromName(stack)

	if err != nil {
		RenderError(rw, err)
		return
	}

	env, err := models.GetEnvironment(app)

	if err != nil {
		RenderError(rw, err)
		return
	}

	// convert Port5432TcpAddr to POSTGRES_PORT_5432_TCP_ADDR
	re := regexp.MustCompile("([a-z])([A-Z0-9])") // lower case letter followed by upper case or number, i.e. Port5432
	re2 := regexp.MustCompile("([0-9])([A-Z])")   // number followed by upper case letter, i.e. 5432Tcp

	for k, v := range s.Outputs {
		u := re.ReplaceAllString(k, "${1}_${2}")
		u = re2.ReplaceAllString(u, "${1}_${2}")
		u = name + "_" + u
		u = strings.ToUpper(u)

		env[u] = v
	}

	err = models.PutEnvironment(app, env)
	service.Save()

	Redirect(rw, r, fmt.Sprintf("/apps/%s#services", app))
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


func servicesLogger(at string) *logger.Logger {
	return logger.New("ns=kernel cn=services").At(at)
}
