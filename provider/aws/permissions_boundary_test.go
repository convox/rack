package aws

import (
	"bytes"
	"encoding/json"
	"html/template"
	"path/filepath"
	"strings"
	"testing"

	"github.com/convox/rack/pkg/manifest"
	"github.com/convox/rack/pkg/manifest1"
	"github.com/convox/rack/pkg/structs"
	"github.com/stretchr/testify/require"
)

const testBoundary = "arn:aws:iam::123456789012:policy/convox-boundary/ceiling"

func renderFormation(t *testing.T, name string, data interface{}) map[string]interface{} {
	t.Helper()

	path := filepath.Join("formation", name+".json.tmpl")
	tpl, err := template.New(filepath.Base(path)).Funcs(formationHelpers()).ParseFiles(path)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, tpl.Execute(&buf, data))

	var v map[string]interface{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &v), buf.String())
	return v
}

func iamEntities(tmpl map[string]interface{}) map[string]map[string]interface{} {
	out := map[string]map[string]interface{}{}
	resources, _ := tmpl["Resources"].(map[string]interface{})
	for name, r := range resources {
		res, _ := r.(map[string]interface{})
		switch res["Type"] {
		case "AWS::IAM::Role", "AWS::IAM::User":
			props, _ := res["Properties"].(map[string]interface{})
			out[name] = props
		}
	}
	return out
}

func withBoundary(data map[string]interface{}, value string) map[string]interface{} {
	out := map[string]interface{}{"PermissionsBoundary": value}
	for k, v := range data {
		out[k] = v
	}
	return out
}

// assertBoundary renders a template without the key, with an empty value, and
// with an ARN. The first two must declare the same resources and the third must
// put the ARN on every IAM entity, including each one listed.
func assertBoundary(t *testing.T, name string, data map[string]interface{}, entities ...string) {
	t.Helper()

	require.Equal(t, renderFormation(t, name, data)["Resources"], renderFormation(t, name, withBoundary(data, ""))["Resources"])

	set := iamEntities(renderFormation(t, name, withBoundary(data, testBoundary)))
	for _, e := range entities {
		require.Contains(t, set, e)
	}
	for e, props := range set {
		require.Equal(t, testBoundary, props["PermissionsBoundary"], e)
	}
}

func gen2Fixture(t *testing.T) *manifest.Manifest {
	t.Helper()

	m, err := manifest.Load([]byte(`services:
  web:
    port: 3000
    policies:
      - arn:aws:iam::aws:policy/ReadOnlyAccess
    scale:
      count: 1-3
      targets:
        cpu: 50
timers:
  cleanup:
    command: bin/cleanup
    schedule: "*/5 * * * ?"
    service: web
    policies:
      - arn:aws:iam::aws:policy/ReadOnlyAccess
`), nil)
	require.NoError(t, err)
	return m
}

func TestPermissionsBoundaryAppTemplate(t *testing.T) {
	m := gen2Fixture(t)
	release := &structs.Release{Id: "R12345", App: "testapp", Build: "B12345"}
	build := &structs.Build{Id: "B12345", App: "testapp"}

	assertBoundary(t, "app", map[string]interface{}{
		"App":      "testapp",
		"Manifest": m,
		"Password": "testpass",
		"Release":  release,
		"Version":  "test",
	}, "ExecutionRole", "ServiceRole", "TimerRole")

	assertBoundary(t, "service", map[string]interface{}{
		"App":           "testapp",
		"Autoscale":     true,
		"Build":         build,
		"DeploymentMin": 50,
		"DeploymentMax": 200,
		"Manifest":      m,
		"Password":      "testpass",
		"Release":       release,
		"Service":       m.Services[0],
		"Tags":          m.Services[0].Tags,
	}, "AutoscalingRole", "ExecutionRole", "DedicatedRole")

	assertBoundary(t, "timer", map[string]interface{}{
		"App":      "testapp",
		"Build":    build,
		"Manifest": m,
		"Password": "testpass",
		"Release":  release,
		"Timer":    m.Timers[0],
	}, "DedicatedRole")
}

func TestPermissionsBoundaryAppCreateTemplate(t *testing.T) {
	require.Equal(t, renderFormation(t, "app", nil)["Resources"], renderFormation(t, "app", map[string]interface{}{"PermissionsBoundary": ""})["Resources"])
	assertBoundary(t, "app", map[string]interface{}{}, "ExecutionRole", "ServiceRole")
}

func TestPermissionsBoundaryGen1Template(t *testing.T) {
	m, err := manifest1.Load([]byte(`web:
  image: httpd
  ports:
    - 80:80
  labels:
    - convox.cron.cleanup=*/5 * * * ? bin/cleanup
`))
	require.NoError(t, err)

	app := &structs.App{Name: "testapp", Parameters: map[string]string{}, Outputs: map[string]string{}}

	assertBoundary(t, "g1/app", map[string]interface{}{
		"App":      app,
		"Build":    &structs.Build{Id: "B12345", App: "testapp"},
		"Manifest": m,
		"Version":  "test",
	}, "CustomTopicRole", "SecureEnvironmentRole", "CronRole", "ServiceRole")

	assertBoundary(t, "g1/app", map[string]interface{}{"Version": "test"}, "CustomTopicRole", "SecureEnvironmentRole", "ServiceRole")
}

func TestPermissionsBoundaryResourceTemplates(t *testing.T) {
	require.True(t, resourceSystemParameters["PermissionsBoundary"])
	require.Contains(t, strings.Split(redactedParams, ","), "PermissionsBoundary")

	want := map[string]interface{}{"Fn::If": []interface{}{"PermissionsBoundaryEnabled", map[string]interface{}{"Ref": "PermissionsBoundary"}, map[string]interface{}{"Ref": "AWS::NoValue"}}}
	enabled := map[string]interface{}{"Fn::Not": []interface{}{map[string]interface{}{"Fn::Equals": []interface{}{map[string]interface{}{"Ref": "PermissionsBoundary"}, ""}}}}

	for name, entity := range map[string]string{"s3": "User", "sns": "User", "sqs": "User", "syslog": "Role", "webhook": "ForwarderRole"} {
		path := filepath.Join("templates", "resource", name+".tmpl")
		tpl, err := template.New(name).Funcs(templateHelpers()).ParseFiles(path)
		require.NoError(t, err)

		var buf bytes.Buffer
		require.NoError(t, tpl.ExecuteTemplate(&buf, "resource", &structs.Resource{}))

		var v map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &v), name)

		params, _ := v["Parameters"].(map[string]interface{})
		param, _ := params["PermissionsBoundary"].(map[string]interface{})
		require.Equal(t, "", param["Default"], name)

		conds, _ := v["Conditions"].(map[string]interface{})
		require.Equal(t, enabled, conds["PermissionsBoundaryEnabled"], name)

		require.Equal(t, want, iamEntities(v)[entity]["PermissionsBoundary"], name)
	}
}
