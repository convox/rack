package controllers

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/websocket"

	"github.com/convox/kernel/helpers"
	"github.com/convox/kernel/models"
)

func init() {
	RegisterPartial("app", "builds")
	RegisterPartial("app", "changes")
	RegisterPartial("app", "debug")
	RegisterPartial("app", "deployments")
	RegisterPartial("app", "environment")
	RegisterPartial("app", "logs")
	RegisterPartial("app", "releases")
	RegisterPartial("app", "resources")

	RegisterPartial("app", "AMI")
	RegisterPartial("app", "AWS::AutoScaling::AutoScalingGroup")
	RegisterPartial("app", "AWS::AutoScaling::LaunchConfiguration")
	RegisterPartial("app", "AWS::CloudFormation::Stack")
	RegisterPartial("app", "AWS::EC2::SecurityGroup")
	RegisterPartial("app", "AWS::EC2::VPC")
	RegisterPartial("app", "AWS::ElasticLoadBalancing::LoadBalancer")
	RegisterPartial("app", "AWS::IAM::InstanceProfile")
	RegisterPartial("app", "AWS::IAM::Role")
	RegisterPartial("app", "AWS::Kinesis::Stream")
	RegisterPartial("app", "AWS::RDS::DBInstance")
	RegisterPartial("app", "AWS::S3::Bucket")
	RegisterPartial("app", "Env::Diff")

	RegisterTemplate("apps", "layout", "apps")
	RegisterTemplate("app", "layout", "app")
}

func AppList(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("list").Start()

	apps, err := models.ListApps()

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	sort.Sort(apps)

	params := map[string]interface{}{
		"Apps": apps,
	}

	switch r.Header.Get("Content-Type") {
	case "application/json":
		RenderJson(rw, apps)
	default:
		RenderTemplate(rw, "apps", params)
	}
}

func AppShow(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("show").Start()

	app := mux.Vars(r)["app"]

	a, err := models.GetApp(app)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	switch r.Header.Get("Content-Type") {
	case "application/json":
		RenderJson(rw, a)
	default:
		RenderTemplate(rw, "app", a)
	}
}

func AppCreate(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("create").Start()

	name := GetForm(r, "name")
	repo := GetForm(r, "repo")

	app := &models.App{
		Name:       name,
		Repository: repo,
	}

	err := app.Create()

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	Redirect(rw, r, fmt.Sprintf("/apps/%s", name))
}

func AppUpdate(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("update").Start()

	vars := mux.Vars(r)
	name := vars["app"]

	app, err := models.GetApp(name)

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	params := map[string]string{}

	if count := GetForm(r, "count"); count != "" {
		params["DesiredCount"] = count
	}

	if cpu := GetForm(r, "cpu"); cpu != "" {
		params["Cpu"] = cpu
	}

	if mem := GetForm(r, "mem"); mem != "" {
		params["Memory"] = mem
	}

	if len(params) > 0 {
		if err := app.UpdateParams(params); err != nil {
			log.Error(err)
			RenderError(rw, err)
			return
		}
	}

	Redirect(rw, r, fmt.Sprintf("/apps/%s", name))
}

func AppDelete(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("delete").Start()

	vars := mux.Vars(r)
	name := vars["app"]

	app, err := models.GetApp(name)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	log.Success("step=app.get app=%q", app.Name)

	err = app.Delete()

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	log.Success("step=app.delete app=%q", app.Name)

	RenderText(rw, "ok")
}

func AppPromote(rw http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	app := vars["app"]

	release, err := models.GetRelease(app, GetForm(r, "release"))

	if err != nil {
		RenderError(rw, err)
		return
	}

	err = release.Promote()

	if err != nil {
		RenderError(rw, err)
		return
	}

	Redirect(rw, r, fmt.Sprintf("/apps/%s", app))
}

func AppChanges(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("changes").Start()

	app := mux.Vars(r)["app"]

	changes, err := models.ListChanges(app)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "changes", changes)
}

func AppDeployments(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("deployments").Start()

	app := mux.Vars(r)["app"]

	deployments, err := models.ListDeployments(app)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "deployments", deployments)
}

func AppEnvironment(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("environment").Start()

	app := mux.Vars(r)["app"]

	env, err := models.GetEnvironment(app)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	params := map[string]interface{}{
		"App":         app,
		"Environment": env,
	}

	switch r.Header.Get("Content-Type") {
	case "application/json":
		RenderJson(rw, params["Environment"])
	default:
		RenderPartial(rw, "app", "environment", params)
	}
}

func AppDebug(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("environment").Start()

	app := mux.Vars(r)["app"]

	a, err := models.GetApp(app)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "debug", a)
}

var regexServiceCleaner = regexp.MustCompile(`\(service ([^)]+)\) (.*)`)

func AppEvents(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("events").Start()

	app := mux.Vars(r)["app"]

	events, err := models.ListECSEvents(app)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	for i, _ := range events {
		match := regexServiceCleaner.FindStringSubmatch(events[i].Message)

		if len(match) == 3 {
			events[i].Message = fmt.Sprintf("[ECS] (%s) %s", match[1], match[2])
		}
	}

	es, err := models.ListEvents(app)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	for _, e := range es {
		events = append(events, models.ServiceEvent{
			Message:   fmt.Sprintf("[CFM] (%s) %s %s", e.Name, e.Status, e.Reason),
			CreatedAt: e.Time,
		})
	}

	sort.Sort(sort.Reverse(events))

	data := ""

	for _, e := range events {
		data += fmt.Sprintf("%s: %s\n", e.CreatedAt.Format(time.RFC3339), e.Message)
	}

	RenderText(rw, data)
}

func AppLogs(rw http.ResponseWriter, r *http.Request) {
	// log := appsLogger("logs").Start()

	app := mux.Vars(r)["app"]

	RenderPartial(rw, "app", "logs", app)
}

func AppNameAvailable(rw http.ResponseWriter, r *http.Request) {
	app, _ := models.GetApp(mux.Vars(r)["app"])

	if app != nil {
		RenderText(rw, "false")
	} else {
		RenderText(rw, "true")
	}
}

func AppStream(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("stream").Start()

	app, err := models.GetApp(mux.Vars(r)["app"])

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	logs := make(chan []byte)
	done := make(chan bool)

	app.SubscribeLogs(logs, done)

	ws, err := upgrader.Upgrade(rw, r, nil)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	log.Success("step=upgrade app=%q", app.Name)

	defer ws.Close()

	for data := range logs {
		ws.WriteMessage(websocket.TextMessage, data)
	}

	log.Success("step=ended app=%q", app.Name)
}

func AppReleases(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("releases").Start()

	vars := mux.Vars(r)
	app := vars["app"]

	l := map[string]string{
		"id":      r.URL.Query().Get("id"),
		"created": r.URL.Query().Get("created"),
	}

	a, err := models.GetApp(app)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	releases, err := models.ListReleases(app, l)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	params := map[string]interface{}{
		"App":      a,
		"Releases": releases,
	}

	if len(releases) > 0 {
		params["Last"] = releases[len(releases)-1]
	}

	RenderPartial(rw, "app", "releases", params)
}

func AppResources(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("resources").Start()

	app := mux.Vars(r)["app"]

	resources, err := models.ListResources(app)

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "resources", resources)
}

func AppStatus(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("status").Start()

	app, err := models.GetApp(mux.Vars(r)["app"])

	if err != nil {
		helpers.Error(log, err)
		RenderError(rw, err)
		return
	}

	RenderText(rw, app.Status)
}

func appsLogger(at string) *logger.Logger {
	return logger.New("ns=kernel cn=apps").At(at)
}
