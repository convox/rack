package controllers

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"time"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/Godeps/_workspace/src/golang.org/x/net/websocket"

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

func AppList(rw http.ResponseWriter, r *http.Request) error {
	apps, err := models.ListApps()

	if err != nil {
		return err
	}

	sort.Sort(apps)

	return RenderJson(rw, apps)
}

func AppShow(rw http.ResponseWriter, r *http.Request) error {
	app := mux.Vars(r)["app"]

	a, err := models.GetApp(mux.Vars(r)["app"])

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
	}

	if err != nil {
		return err
	}

	return RenderJson(rw, a)
}

func AppCreate(rw http.ResponseWriter, r *http.Request) error {
	name := GetForm(r, "name")

	app := &models.App{
		Name: name,
	}

	err := app.Create()

	if awsError(err) == "AlreadyExistsException" {
		app, err := models.GetApp(name)

		if err != nil {
			return err
		}

		return RenderForbidden(rw, fmt.Sprintf("There is already an app named %s (%s)", name, app.Status))
	}

	if err != nil {
		return err
	}

	app, err = models.GetApp(name)

	if err != nil {
		return err
	}

	return RenderJson(rw, app)
}

func AppDelete(rw http.ResponseWriter, r *http.Request) error {
	name := mux.Vars(r)["app"]

	app, err := models.GetApp(name)

	if awsError(err) == "ValidationError" {
		return RenderNotFound(rw, fmt.Sprintf("no such app: %s", name))
	}

	if err != nil {
		return err
	}

	err = app.Delete()

	if err != nil {
		return err
	}

	return RenderSuccess(rw)
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

	if awsError(err) == "ValidationError" {
		RenderNotFound(rw, fmt.Sprintf("no such app: %s", app))
		return
	}

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

func AppLogs(ws *websocket.Conn) error {
	defer ws.Close()

	app := mux.Vars(ws.Request())["app"]

	a, err := models.GetApp(app)

	if awsError(err) == "ValidationError" {
		return fmt.Errorf("no such app: %s", app)
	}

	if err != nil {
		return err
	}

	logs := make(chan []byte)
	done := make(chan bool)

	a.SubscribeLogs(logs, done)

	for data := range logs {
		ws.Write(data)
	}

	return nil
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

	switch r.Header.Get("Content-Type") {
	case "application/json":
		RenderJson(rw, releases)
	default:
		RenderPartial(rw, "app", "releases", params)
	}
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
