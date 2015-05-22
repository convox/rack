package controllers

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/convox/kernel/Godeps/_workspace/src/github.com/ddollar/logger"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/convox/kernel/Godeps/_workspace/src/github.com/gorilla/websocket"

	"github.com/convox/kernel/models"
)

func init() {
	RegisterPartial("app", "builds")
	RegisterPartial("app", "changes")
	RegisterPartial("app", "environment")
	RegisterPartial("app", "logs")
	RegisterPartial("app", "releases")
	RegisterPartial("app", "resources")
	RegisterPartial("app", "services")

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
		log.Error(err)
		RenderError(rw, err)
		return
	}

	sort.Sort(apps)

	clusters, err := models.ListClusters()

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	params := map[string]interface{}{
		"Apps":     apps,
		"Clusters": clusters,
	}

	RenderTemplate(rw, "apps", params)
}

func AppShow(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("show").Start()

	app := mux.Vars(r)["app"]

	a, err := models.GetApp(app)

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	RenderTemplate(rw, "app", a)
}

func AppCreate(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("create").Start()

	cluster := GetForm(r, "cluster")
	name := GetForm(r, "name")
	repo := GetForm(r, "repo")

	app := &models.App{
		Cluster:    cluster,
		Name:       name,
		Repository: repo,
	}

	err := app.Create()

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	Redirect(rw, r, fmt.Sprintf("/apps/%s", name))
}

func AppDelete(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("delete").Start()

	vars := mux.Vars(r)
	name := vars["app"]

	app, err := models.GetApp(name)

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	log.Success("step=app.get app=%q", app.Name)

	err = app.Delete()

	if err != nil {
		log.Error(err)
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

	// change := &models.Change{
	//   App:      app,
	//   Created:  time.Now(),
	//   Metadata: "{}",
	//   TargetId: release.Id,
	//   Type:     "PROMOTE",
	//   Status:   "changing",
	//   User:     "convox",
	// }

	// change.Save()

	// events, err := models.ListEvents(app)
	// if err != nil {
	//   log.Error(err)
	//   change.Status = "failed"
	//   change.Metadata = err.Error()
	//   change.Save()

	//   RenderError(rw, err)
	//   return
	// }
	// log.Success("step=events.list app=%q release=%q", release.App, release.Id)

	err = release.Promote()

	if err != nil {
		RenderError(rw, err)
		return
	}

	// if err != nil {
	//   log.Error(err)
	//   change.Status = "failed"
	//   change.Metadata = fmt.Sprintf("{\"error\": \"%s\"}", err.Error())
	//   change.Save()

	//   RenderError(rw, err)
	//   return
	// }
	// log.Success("step=release.promote app=%q release=%q", release.App, release.Id)

	Redirect(rw, r, fmt.Sprintf("/apps/%s", app))

	// a, err := models.GetApp("", app)
	// if err != nil {
	//   log.Error(err)
	//   panic(err)
	// }
	// log.Success("step=app.get app=%q", release.App)
	// go a.WatchForCompletion(change, events)
}

func AppBuilds(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("builds").Start()

	vars := mux.Vars(r)
	app := vars["app"]

	builds, err := models.ListBuilds(app)

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "builds", builds)
}

func AppChanges(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("changes").Start()

	app := mux.Vars(r)["app"]

	changes, err := models.ListChanges(app)

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "changes", changes)
}

func AppEnvironment(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("environment").Start()

	app := mux.Vars(r)["app"]

	env, err := models.GetEnvironment(app)

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	params := map[string]interface{}{
		"App":         app,
		"Environment": env,
	}

	RenderPartial(rw, "app", "environment", params)
}

func AppLogs(rw http.ResponseWriter, r *http.Request) {
	// log := appsLogger("logs").Start()

	app := mux.Vars(r)["app"]

	RenderPartial(rw, "app", "logs", app)
}

func AppStream(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("stream").Start()

	app, err := models.GetApp(mux.Vars(r)["app"])

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	logs := make(chan []byte)
	done := make(chan bool)

	app.SubscribeLogs(logs, done)

	ws, err := upgrader.Upgrade(rw, r, nil)

	if err != nil {
		log.Error(err)
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

	releases, err := models.ListReleases(app)

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "releases", releases)
}

func AppResources(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("resources").Start()

	app := mux.Vars(r)["app"]

	resources, err := models.ListResources(app)

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "resources", resources)
}

func AppServices(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("services").Start()

	app := mux.Vars(r)["app"]

	services, err := models.ListServices(app)

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	RenderPartial(rw, "app", "services", services)
}

func AppStatus(rw http.ResponseWriter, r *http.Request) {
	log := appsLogger("status").Start()

	app, err := models.GetApp(mux.Vars(r)["app"])

	if err != nil {
		log.Error(err)
		RenderError(rw, err)
		return
	}

	RenderText(rw, app.Status)
}

func appsLogger(at string) *logger.Logger {
	return logger.New("ns=kernel cn=apps").At(at)
}
