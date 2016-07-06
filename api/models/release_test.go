package models

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/aryann/difflib"
	"github.com/convox/rack/test"
	"github.com/stretchr/testify/require"
)

func Diff(t *testing.T, name, s1, s2 string) {
	diff1 := strings.Split(strings.TrimSpace(s1), "\n")
	diff2 := strings.Split(strings.TrimSpace(s2), "\n")

	diff := difflib.Diff(diff1, diff2)
	diffs := []string{}

	// bigger than max
	prev := 1000000

	for l, d := range diff {
		switch d.Delta {
		case difflib.LeftOnly:
			if (l - prev) > 1 {
				diffs = append(diffs, "")
			}
			diffs = append(diffs, fmt.Sprintf("%04d - %s", l, d.Payload))
			prev = l
		case difflib.RightOnly:
			if (l - prev) > 1 {
				diffs = append(diffs, "")
			}
			diffs = append(diffs, fmt.Sprintf("%04d + %s", l, d.Payload))
			prev = l
		}
	}

	if len(diffs) > 0 {
		t.Errorf("Unexpected results for %s:\n%s", name, strings.Join(diffs, "\n"))
	}
}

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
