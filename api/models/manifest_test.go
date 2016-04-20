package models

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/aryann/difflib"
	"github.com/stretchr/testify/assert"
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
		fmt.Println(s2)
	}
}

type Cases []struct {
	got, want interface{}
}

func assertFixture(t *testing.T, name string, primary string) {
	data, err := ioutil.ReadFile(fmt.Sprintf("fixtures/%s.yml", name))
	require.Nil(t, err)

	app := &App{
		Name: "httpd",
		Tags: map[string]string{
			"Name":   "httpd",
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox-test",
		},
	}
	manifest, err := LoadManifest(string(data), app)

	if err != nil {
		fmt.Printf("ERROR: %+v\n", err)
	}

	require.Nil(t, err)

	formation, err := manifest.Formation()

	if err != nil {
		fmt.Printf("ERROR: %+v\n", err)
	}

	require.Nil(t, err)

	data, err = ioutil.ReadFile(fmt.Sprintf("fixtures/%s.json", name))
	require.Nil(t, err)

	Diff(t, name, string(data), formation)
}

func assertFixtureUnbound(t *testing.T, name string, primary string) {
	data, err := ioutil.ReadFile(fmt.Sprintf("fixtures/%s.yml", name))
	require.Nil(t, err)

	app := &App{
		Name: "httpd",
		Tags: map[string]string{
			"Type":   "app",
			"System": "convox",
			"Rack":   "convox-test",
		},
	}
	manifest, err := LoadManifest(string(data), app)

	if err != nil {
		fmt.Printf("ERROR: %+v\n", err)
	}

	require.Nil(t, err)

	for i, _ := range manifest {
		if manifest[i].Name == primary {
			manifest[i].primary = true
		}
	}

	formation, err := manifest.Formation()

	if err != nil {
		fmt.Printf("ERROR: %+v\n", err)
	}

	require.Nil(t, err)

	data, err = ioutil.ReadFile(fmt.Sprintf("fixtures/%s_unbound.json", name))
	require.Nil(t, err)

	Diff(t, name, string(data), formation)
}

func TestManifestInvalid(t *testing.T) {
	manifest, err := LoadManifest("invalid-manifest", nil)

	assert.Nil(t, manifest)
	assert.NotNil(t, err)
	assert.Regexp(t, "^invalid manifest: ", err.Error())
}

func TestManifestFixtures(t *testing.T) {
	os.Setenv("AWS_REGION", "us-test-2")
	ManifestRandomPorts = false
	assertFixture(t, "multi_balancer", "web")
	assertFixture(t, "web_external_internal", "")
	assertFixture(t, "web_postgis", "")
	assertFixture(t, "web_postgis_internal", "")
	assertFixture(t, "worker", "")
	assertFixture(t, "complex_environment", "")
	assertFixture(t, "balancer_labels", "")
	ManifestRandomPorts = true
}

// test unbound apps with old balancer names and primary process logic
func TestManifestFixtureUnbound(t *testing.T) {
	os.Setenv("AWS_REGION", "us-test-2")
	ManifestRandomPorts = false
	assertFixtureUnbound(t, "multi_balancer", "web")
	assertFixtureUnbound(t, "web_external_internal", "")
	assertFixtureUnbound(t, "web_postgis", "")
	assertFixtureUnbound(t, "web_postgis_internal", "")
	assertFixtureUnbound(t, "worker", "")
	ManifestRandomPorts = true
}

func TestCommandString(t *testing.T) {
	entry := ManifestEntry{
		Command: "foo bar baz",
	}
	assert.EqualValues(t, "foo bar baz", entry.CommandString())
	assert.EqualValues(t, []string{}, entry.CommandArray())
}

func TestCommandArray(t *testing.T) {
	var commandSlice []string = []string{"foo", "bar", "baz"}
	var interfaceSlice []interface{} = make([]interface{}, len(commandSlice))
	for i, d := range commandSlice {
		interfaceSlice[i] = d
	}
	entry := ManifestEntry{
		Command: interfaceSlice,
	}
	assert.EqualValues(t, "", entry.CommandString())
	assert.EqualValues(t, []string{"foo", "bar", "baz"}, entry.CommandArray())
}

func TestCommandExecForm(t *testing.T) {
	assertFixture(t, "command_exec_form", "")
}

func TestCommandStringForm(t *testing.T) {
	assertFixture(t, "command_string_form", "")
}

func TestManifestRandomPorts(t *testing.T) {
	manifest, err := LoadManifest("web:\n  ports:\n  - 80:3000\n  - 3001", nil)

	require.Nil(t, err)

	// kinda hacky but just making sure we're not in sequence here
	assert.NotEqual(t, 1, (manifest[0].randoms["3001"] - manifest[0].randoms["80:3000"]))
}

func TestLoadBalancerNameUniquePerEntryWithTruncation(t *testing.T) {
	mb1 := ManifestBalancer{
		Entry: ManifestEntry{
			app: &App{
				Name: "myverylogappname-production",
			},
			Name: "web",
		},
		Public: true,
	}

	mb2 := ManifestBalancer{
		Entry: ManifestEntry{
			app: &App{
				Name: "myverylogappname-production",
			},
			Name: "worker",
		},
		Public: true,
	}

	assert.EqualValues(t, `"myverylogappname-product-DIVTGA7"`, mb1.LoadBalancerName())
	assert.EqualValues(t, `"myverylogappname-product-LQYILNJ"`, mb2.LoadBalancerName())

	assert.Equal(t, 34, len(mb1.LoadBalancerName())) // ELB name is max 32 characters + quotes

	mb1.Public = false
	mb2.Public = false

	assert.EqualValues(t, `"myverylogappname-produ-DIVTGA7-i"`, mb1.LoadBalancerName())
	assert.EqualValues(t, `"myverylogappname-produ-LQYILNJ-i"`, mb2.LoadBalancerName())

	assert.Equal(t, 34, len(mb1.LoadBalancerName())) // ELB name is max 32 characters + quotes
}

func TestLoadBalancerNameUniquePerRack(t *testing.T) {
	// reset RACK after this test
	r := os.Getenv("RACK")
	defer os.Setenv("RACK", r)

	mb := ManifestBalancer{
		Entry: ManifestEntry{
			app: &App{
				Name: "foo",
			},
			Name: "web",
		},
	}

	os.Setenv("RACK", "staging")
	assert.EqualValues(t, `"foo-web-GSAGMQZ-i"`, mb.LoadBalancerName())

	os.Setenv("RACK", "production")
	assert.EqualValues(t, `"foo-web-7MS5NPT-i"`, mb.LoadBalancerName())
}

func TestLoadBalancerNameUnbound(t *testing.T) {
	// an app stack with Tags but no "Name" tag is an unbound/legacy app
	mb := ManifestBalancer{
		Entry: ManifestEntry{
			app: &App{
				Name: "foo",
				Tags: map[string]string{
					"Rack":   "convox",
					"System": "convox",
					"Type":   "app",
				},
			},
			Name:    "web",
			primary: true,
		},
	}

	// legacy naming for backwards compatibility
	assert.Equal(t, `{ "Ref": "AWS::StackName" }`, string(mb.LoadBalancerName()))

	// known bug in primary / internal naming
	mb.Public = false
	assert.Equal(t, `{ "Ref": "AWS::StackName" }`, string(mb.LoadBalancerName()))

	mb.Entry.primary = false
	assert.Equal(t, `{ "Fn::Join": [ "-", [ { "Ref": "AWS::StackName" }, "web", "i" ] ] }`, string(mb.LoadBalancerName()))

	mb.Public = true
	assert.Equal(t, `{ "Fn::Join": [ "-", [ { "Ref": "AWS::StackName" }, "web" ] ] }`, string(mb.LoadBalancerName()))
}
