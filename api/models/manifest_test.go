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
	assertFixture(t, "environment_map", "")
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

func TestHealthCheckPort(t *testing.T) {
	_manifest := `
web:
  ports:
    - 80:3000
    - 81:3001
`
	manifest, err := LoadManifest(_manifest, nil)
	require.Nil(t, err)
	balancer := manifest.Balancers()[0]

	// Should be the first port
	port, err := balancer.HealthCheckPort()
	assert.EqualValues(t, port, "80")
}

func TestHealthCheckPortWithOverride(t *testing.T) {
	_manifest := `
web:
  ports:
    - 80:3000
    - 81:3001
  labels:
    - convox.health_check.port=3001
`
	manifest, err := LoadManifest(_manifest, nil)
	require.Nil(t, err)
	balancer := manifest.Balancers()[0]

	// Should be the first port
	port, err := balancer.HealthCheckPort()
	assert.EqualValues(t, port, "81")
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
