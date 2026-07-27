package aws

import (
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"
	"testing"
)

// TestServiceTemplateParses verifies service.json.tmpl parses cleanly with the
// helper funcs registered in formationHelpers(). This catches template-syntax
// regressions (missing {{ end }}, unknown function names, malformed actions)
// without requiring a full render fixture.
//
// End-to-end rendering is exercised in integration tests.
func TestServiceTemplateParses(t *testing.T) {
	path := filepath.Join("formation", "service.json.tmpl")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	if _, err := template.New(filepath.Base(path)).Funcs(formationHelpers()).Parse(string(data)); err != nil {
		t.Fatalf("service.json.tmpl failed to parse: %v", err)
	}
}

// TestAppTemplateParses is a companion parse check for app.json.tmpl so future
// refactors to formationHelpers or template syntax are caught in both places.
func TestAppTemplateParses(t *testing.T) {
	path := filepath.Join("formation", "app.json.tmpl")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	if _, err := template.New(filepath.Base(path)).Funcs(formationHelpers()).Parse(string(data)); err != nil {
		t.Fatalf("app.json.tmpl failed to parse: %v", err)
	}
}

// TestBuildLogDriverWiring pins what BuildLogDriver depends on: LogGroup must
// exist whenever the build awslogs branch refs it, and the parameter needs an
// unconditional reference or CloudFormation reports no updates on a change.
func TestBuildLogDriverWiring(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("formation", "rack.json"))
	if err != nil {
		t.Fatalf("read rack.json: %v", err)
	}

	type fnIf struct {
		If []json.RawMessage `json:"Fn::If"`
	}

	var tmpl struct {
		Outputs map[string]struct {
			Condition string `json:"Condition"`
		} `json:"Outputs"`
		Resources map[string]struct {
			Condition  string `json:"Condition"`
			Properties struct {
				ContainerDefinitions []struct {
					Environment []struct {
						Name  string          `json:"Name"`
						Value json.RawMessage `json:"Value"`
					} `json:"Environment"`
					LogConfiguration fnIf `json:"LogConfiguration"`
				} `json:"ContainerDefinitions"`
			} `json:"Properties"`
		} `json:"Resources"`
	}

	if err := json.Unmarshal(data, &tmpl); err != nil {
		t.Fatalf("parse rack.json: %v", err)
	}

	if got := tmpl.Resources["LogGroup"].Condition; got != "LogGroupEnabled" {
		t.Errorf("LogGroup resource condition is %q, want LogGroupEnabled", got)
	}

	if got := tmpl.Outputs["LogGroup"].Condition; got != "LogGroupEnabled" {
		t.Errorf("LogGroup output condition is %q, want LogGroupEnabled", got)
	}

	// Only the build task defs follow BuildLogDriver, the rack api tasks stay on
	// the rack-wide driver.
	conditions := map[string][2]string{
		"ApiBuildTasks":   {"BuildLogSyslog", "LogGroupEnabled"},
		"ApiBuildFargate": {"BuildLogSyslog", "LogGroupEnabled"},
		"ApiMonitorTasks": {"EnableSyslog", "EnableCloudWatch"},
		"ApiWebTasks":     {"EnableSyslog", "EnableCloudWatch"},
	}

	for name, want := range conditions {
		cds := tmpl.Resources[name].Properties.ContainerDefinitions
		if len(cds) == 0 {
			t.Fatalf("%s has no container definitions", name)
		}

		var outer string
		var inner fnIf

		if len(cds[0].LogConfiguration.If) != 3 {
			t.Fatalf("%s LogConfiguration is not an Fn::If with three elements", name)
		}
		if err := json.Unmarshal(cds[0].LogConfiguration.If[0], &outer); err != nil {
			t.Fatalf("%s outer condition: %v", name, err)
		}
		if err := json.Unmarshal(cds[0].LogConfiguration.If[2], &inner); err != nil {
			t.Fatalf("%s inner branch: %v", name, err)
		}

		var innerCond string
		if len(inner.If) != 3 {
			t.Fatalf("%s inner branch is not an Fn::If with three elements", name)
		}
		if err := json.Unmarshal(inner.If[0], &innerCond); err != nil {
			t.Fatalf("%s inner condition: %v", name, err)
		}

		if outer != want[0] || innerCond != want[1] {
			t.Errorf("%s log conditions are (%s, %s), want (%s, %s)", name, outer, innerCond, want[0], want[1])
		}
	}

	for _, name := range []string{"ApiBuildTasks", "ApiBuildFargate"} {
		found := false

		for _, e := range tmpl.Resources[name].Properties.ContainerDefinitions[0].Environment {
			if e.Name != "BUILD_LOG_DRIVER" {
				continue
			}

			var ref map[string]string
			if err := json.Unmarshal(e.Value, &ref); err == nil && ref["Ref"] == "BuildLogDriver" {
				found = true
			}
		}

		if !found {
			t.Errorf("%s is missing an unconditional BUILD_LOG_DRIVER environment ref", name)
		}
	}
}
