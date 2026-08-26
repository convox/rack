package aws

import (
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
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

// TestBuildCachePruneWiring pins the ApiBuildTasks environment entry that carries
// PruneOlderImagesInHour into the build container. A wrong name or ref there leaves
// the build cache unpruned with no other symptom.
func TestBuildCachePruneWiring(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("formation", "rack.json"))
	if err != nil {
		t.Fatalf("read rack.json: %v", err)
	}

	var tmpl struct {
		Resources map[string]struct {
			Properties struct {
				ContainerDefinitions []struct {
					Environment []struct {
						Name  string          `json:"Name"`
						Value json.RawMessage `json:"Value"`
					} `json:"Environment"`
				} `json:"ContainerDefinitions"`
			} `json:"Properties"`
		} `json:"Resources"`
	}

	if err := json.Unmarshal(data, &tmpl); err != nil {
		t.Fatalf("parse rack.json: %v", err)
	}

	cds := tmpl.Resources["ApiBuildTasks"].Properties.ContainerDefinitions
	if len(cds) == 0 {
		t.Fatalf("ApiBuildTasks has no container definitions")
	}

	found := false

	for _, e := range cds[0].Environment {
		if e.Name != "BUILD_CACHE_PRUNE_HOURS" {
			continue
		}

		var ref map[string]string
		if err := json.Unmarshal(e.Value, &ref); err == nil && ref["Ref"] == "PruneOlderImagesInHour" {
			found = true
		}
	}

	if !found {
		t.Errorf("ApiBuildTasks is missing a BUILD_CACHE_PRUNE_HOURS ref to PruneOlderImagesInHour")
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

// TestFargateBuildNofileUlimit pins the descriptor headroom on the Fargate build
// task, which the EC2 builders get from the docker daemon default instead. A
// missing or lowered value only shows up as a failed build on a large source tree.
func TestFargateBuildNofileUlimit(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("formation", "rack.json"))
	if err != nil {
		t.Fatalf("read rack.json: %v", err)
	}

	var tmpl struct {
		Resources map[string]struct {
			Properties struct {
				ContainerDefinitions []struct {
					Ulimits []struct {
						Name      string `json:"Name"`
						SoftLimit string `json:"SoftLimit"`
						HardLimit string `json:"HardLimit"`
					} `json:"Ulimits"`
				} `json:"ContainerDefinitions"`
			} `json:"Properties"`
		} `json:"Resources"`
	}

	if err := json.Unmarshal(data, &tmpl); err != nil {
		t.Fatalf("parse rack.json: %v", err)
	}

	cds := tmpl.Resources["ApiBuildFargate"].Properties.ContainerDefinitions
	if len(cds) == 0 {
		t.Fatalf("ApiBuildFargate has no container definitions")
	}

	found := false

	for _, u := range cds[0].Ulimits {
		if u.Name != "nofile" {
			continue
		}

		found = true

		if u.SoftLimit != "1024000" || u.HardLimit != "1024000" {
			t.Errorf("ApiBuildFargate nofile ulimit is (%s, %s), want (1024000, 1024000)", u.SoftLimit, u.HardLimit)
		}
	}

	if !found {
		t.Errorf("ApiBuildFargate is missing a nofile ulimit")
	}
}

// TestFargateBuildCpuMemoryFallback pins the blank-value fallback on the Fargate
// build task definition. A wrong literal fails task definition registration, and
// each parameter needs an unconditional ref or CloudFormation reports no updates.
func TestFargateBuildCpuMemoryFallback(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("formation", "rack.json"))
	if err != nil {
		t.Fatalf("read rack.json: %v", err)
	}

	var tmpl struct {
		Conditions map[string]json.RawMessage `json:"Conditions"`
		Resources  map[string]struct {
			Properties map[string]json.RawMessage `json:"Properties"`
		} `json:"Resources"`
	}

	if err := json.Unmarshal(data, &tmpl); err != nil {
		t.Fatalf("parse rack.json: %v", err)
	}

	props := tmpl.Resources["ApiBuildFargate"].Properties

	var cds []struct {
		Environment []struct {
			Name  string          `json:"Name"`
			Value json.RawMessage `json:"Value"`
		} `json:"Environment"`
	}

	if err := json.Unmarshal(props["ContainerDefinitions"], &cds); err != nil {
		t.Fatalf("ApiBuildFargate container definitions: %v", err)
	}

	if len(cds) == 0 {
		t.Fatalf("ApiBuildFargate has no container definitions")
	}

	fallbacks := []struct {
		property  string
		condition string
		fallback  string
		parameter string
		env       string
	}{
		{"Cpu", "BlankFargateBuildCpu", "1024", "FargateBuildCpu", "FARGATE_BUILD_CPU"},
		{"Memory", "BlankFargateBuildMemory", "4096", "FargateBuildMemory", "FARGATE_BUILD_MEMORY"},
	}

	for _, f := range fallbacks {
		var fn struct {
			If []json.RawMessage `json:"Fn::If"`
		}

		if err := json.Unmarshal(props[f.property], &fn); err != nil {
			t.Fatalf("ApiBuildFargate %s: %v", f.property, err)
		}

		if len(fn.If) != 3 {
			t.Fatalf("ApiBuildFargate %s is not an Fn::If with three elements", f.property)
		}

		var condition, fallback string
		var override map[string]string

		if err := json.Unmarshal(fn.If[0], &condition); err != nil {
			t.Fatalf("%s condition: %v", f.property, err)
		}
		if err := json.Unmarshal(fn.If[1], &fallback); err != nil {
			t.Fatalf("%s fallback: %v", f.property, err)
		}
		if err := json.Unmarshal(fn.If[2], &override); err != nil {
			t.Fatalf("%s override: %v", f.property, err)
		}

		if condition != f.condition || fallback != f.fallback || override["Ref"] != f.parameter {
			t.Errorf("%s is (%s, %s, %s), want (%s, %s, %s)", f.property, condition, fallback, override["Ref"], f.condition, f.fallback, f.parameter)
		}

		var eq struct {
			Equals []json.RawMessage `json:"Fn::Equals"`
		}

		if err := json.Unmarshal(tmpl.Conditions[f.condition], &eq); err != nil {
			t.Fatalf("condition %s: %v", f.condition, err)
		}

		if len(eq.Equals) != 2 {
			t.Fatalf("condition %s is not an Fn::Equals with two elements", f.condition)
		}

		var ref map[string]string
		var blank string

		if err := json.Unmarshal(eq.Equals[0], &ref); err != nil {
			t.Fatalf("condition %s ref: %v", f.condition, err)
		}
		if err := json.Unmarshal(eq.Equals[1], &blank); err != nil {
			t.Fatalf("condition %s comparand: %v", f.condition, err)
		}

		if ref["Ref"] != f.parameter || blank != "" {
			t.Errorf("condition %s compares %s to %q, want %s to empty", f.condition, ref["Ref"], blank, f.parameter)
		}

		found := false

		for _, e := range cds[0].Environment {
			if e.Name != f.env {
				continue
			}

			var eref map[string]string
			if err := json.Unmarshal(e.Value, &eref); err == nil && eref["Ref"] == f.parameter {
				found = true
			}
		}

		if !found {
			t.Errorf("ApiBuildFargate is missing an unconditional %s ref to %s", f.env, f.parameter)
		}
	}
}

// TestFargateBuildEphemeralStorage pins FargateBuildVolumeSize: the blank default,
// the pattern that holds it to whole numbers in range, and the Fn::If that omits
// EphemeralStorage when blank. Drift resizes the build disk or fails the update.
func TestFargateBuildEphemeralStorage(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("formation", "rack.json"))
	if err != nil {
		t.Fatalf("read rack.json: %v", err)
	}

	var tmpl struct {
		Parameters map[string]json.RawMessage `json:"Parameters"`
		Conditions map[string]json.RawMessage `json:"Conditions"`
		Resources  map[string]struct {
			Properties map[string]json.RawMessage `json:"Properties"`
		} `json:"Resources"`
	}

	if err := json.Unmarshal(data, &tmpl); err != nil {
		t.Fatalf("parse rack.json: %v", err)
	}

	rawParam, ok := tmpl.Parameters["FargateBuildVolumeSize"]
	if !ok {
		t.Fatalf("rack.json has no FargateBuildVolumeSize parameter")
	}

	var param struct {
		Type           string          `json:"Type"`
		Default        json.RawMessage `json:"Default"`
		AllowedPattern string          `json:"AllowedPattern"`
	}

	if err := json.Unmarshal(rawParam, &param); err != nil {
		t.Fatalf("parse FargateBuildVolumeSize: %v", err)
	}

	if param.Type != "String" {
		t.Errorf("FargateBuildVolumeSize Type is %q, want String", param.Type)
	}

	if len(param.Default) == 0 {
		t.Fatalf("FargateBuildVolumeSize has no Default")
	}

	var def string
	if err := json.Unmarshal(param.Default, &def); err != nil {
		t.Fatalf("FargateBuildVolumeSize Default must be a JSON string, got %s", param.Default)
	}

	if def != "" {
		t.Errorf("FargateBuildVolumeSize Default is %q, want blank", def)
	}

	if param.AllowedPattern == "" {
		t.Errorf("FargateBuildVolumeSize has no AllowedPattern")
	} else {
		re, err := regexp.Compile(param.AllowedPattern)
		if err != nil {
			t.Fatalf("FargateBuildVolumeSize AllowedPattern does not compile: %v", err)
		}

		for _, v := range []string{"", "21", "50", "100", "200"} {
			if !re.MatchString(v) {
				t.Errorf("AllowedPattern rejects %q, want it accepted", v)
			}
		}

		for _, v := range []string{"20", "20.0", "50.5", "10", "201", "500", "1050", "abc"} {
			if re.MatchString(v) {
				t.Errorf("AllowedPattern accepts %q, want it rejected", v)
			}
		}
	}

	rawCond, ok := tmpl.Conditions["BlankFargateBuildVolumeSize"]
	if !ok {
		t.Fatalf("rack.json has no BlankFargateBuildVolumeSize condition")
	}

	var cond struct {
		Equals []json.RawMessage `json:"Fn::Equals"`
	}

	if err := json.Unmarshal(rawCond, &cond); err != nil {
		t.Fatalf("parse BlankFargateBuildVolumeSize: %v", err)
	}

	if len(cond.Equals) != 2 {
		t.Fatalf("BlankFargateBuildVolumeSize is not an Fn::Equals with two elements")
	}

	var condRef map[string]string
	if err := json.Unmarshal(cond.Equals[0], &condRef); err != nil || condRef["Ref"] != "FargateBuildVolumeSize" {
		t.Errorf("BlankFargateBuildVolumeSize tests %s, want a ref to FargateBuildVolumeSize", cond.Equals[0])
	}

	var sentinel string
	if err := json.Unmarshal(cond.Equals[1], &sentinel); err != nil {
		t.Errorf("BlankFargateBuildVolumeSize sentinel is not a JSON string: %s", cond.Equals[1])
	} else if sentinel != def {
		t.Errorf("BlankFargateBuildVolumeSize sentinel is %q, want the parameter default %q", sentinel, def)
	}

	rawStorage, ok := tmpl.Resources["ApiBuildFargate"].Properties["EphemeralStorage"]
	if !ok {
		t.Fatalf("ApiBuildFargate has no EphemeralStorage property")
	}

	var storage struct {
		If []json.RawMessage `json:"Fn::If"`
	}

	if err := json.Unmarshal(rawStorage, &storage); err != nil {
		t.Fatalf("parse ApiBuildFargate EphemeralStorage: %v", err)
	}

	if len(storage.If) != 3 {
		t.Fatalf("ApiBuildFargate EphemeralStorage is not an Fn::If with three elements")
	}

	var storageCond string
	if err := json.Unmarshal(storage.If[0], &storageCond); err != nil {
		t.Fatalf("ApiBuildFargate EphemeralStorage condition: %v", err)
	}

	if storageCond != "BlankFargateBuildVolumeSize" {
		t.Errorf("ApiBuildFargate EphemeralStorage is gated on %q, want BlankFargateBuildVolumeSize", storageCond)
	}

	var blankBranch map[string]string
	if err := json.Unmarshal(storage.If[1], &blankBranch); err != nil || blankBranch["Ref"] != "AWS::NoValue" {
		t.Errorf("ApiBuildFargate EphemeralStorage blank branch is %s, want a ref to AWS::NoValue", storage.If[1])
	}

	var sizedBranch map[string]json.RawMessage
	if err := json.Unmarshal(storage.If[2], &sizedBranch); err != nil {
		t.Fatalf("parse ApiBuildFargate EphemeralStorage sized branch: %v", err)
	}

	rawSize, ok := sizedBranch["SizeInGiB"]
	if !ok || len(sizedBranch) != 1 {
		t.Errorf("ApiBuildFargate EphemeralStorage sized branch is %s, want exactly SizeInGiB", storage.If[2])
	} else {
		var sizeRef map[string]string
		if err := json.Unmarshal(rawSize, &sizeRef); err != nil || sizeRef["Ref"] != "FargateBuildVolumeSize" {
			t.Errorf("ApiBuildFargate EphemeralStorage SizeInGiB is %s, want a ref to FargateBuildVolumeSize", rawSize)
		}
	}

	if _, ok := tmpl.Resources["ApiBuildTasks"].Properties["EphemeralStorage"]; ok {
		t.Errorf("ApiBuildTasks carries EphemeralStorage, which is a Fargate-only property")
	}

	rawDefs, ok := tmpl.Resources["ApiBuildTasks"].Properties["ContainerDefinitions"]
	if !ok {
		t.Fatalf("ApiBuildTasks has no ContainerDefinitions property")
	}

	var containers []struct {
		Environment []struct {
			Name  string          `json:"Name"`
			Value json.RawMessage `json:"Value"`
		} `json:"Environment"`
	}

	if err := json.Unmarshal(rawDefs, &containers); err != nil {
		t.Fatalf("parse ApiBuildTasks container definitions: %v", err)
	}

	if len(containers) == 0 {
		t.Fatalf("ApiBuildTasks has no container definitions")
	}

	anchors := map[string]string{
		"FARGATE_BUILD_VOLUME_SIZE": "FargateBuildVolumeSize",
		"FARGATE_BUILD_CPU":         "FargateBuildCpu",
		"FARGATE_BUILD_MEMORY":      "FargateBuildMemory",
	}

	for _, e := range containers[0].Environment {
		param, ok := anchors[e.Name]
		if !ok {
			continue
		}

		var ref map[string]string
		if err := json.Unmarshal(e.Value, &ref); err == nil && ref["Ref"] == param {
			delete(anchors, e.Name)
		}
	}

	for name, param := range anchors {
		t.Errorf("ApiBuildTasks is missing an unconditional %s ref to %s", name, param)
	}
}
