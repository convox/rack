package models

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/convox/rack/test"
	"github.com/stretchr/testify/require"
)

func TestLinks(t *testing.T) {
	t.Skip("skipping until we have a strategy for stubbing out the registry dependency")

	os.Setenv("RACK", "convox-test")
	os.Setenv("CLUSTER", "convox-test")

	resp, err := ioutil.ReadFile("fixtures/get-app-template-response.xml")
	require.Nil(t, err)

	fixData, err := ioutil.ReadFile("fixtures/web_redis.json")
	require.Nil(t, err)

	yamlData, err := ioutil.ReadFile("fixtures/web_redis.yml")
	require.Nil(t, err)

	getAppTemplateCycle := test.GetAppTemplateCycle("web")
	getAppTemplateCycle.Response.Body = string(resp)
	stubAws := test.StubAws(
		getAppTemplateCycle,
		test.DescribeAppStackCycle("web"),
	)
	defer stubAws.Close()

	release := &Release{
		Id:       "DEADBEEF",
		App:      "web",
		Build:    "DEADBEEF",
		Env:      "",
		Manifest: string(yamlData),
	}

	ManifestRandomPorts = false
	formation, err := release.Formation()
	require.Nil(t, err)
	ManifestRandomPorts = true

	Diff(t, "web_redis", string(fixData), formation)
}
