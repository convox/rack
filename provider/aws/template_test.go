package aws

import (
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
