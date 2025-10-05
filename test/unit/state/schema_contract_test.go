package statecontracts_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func loadStateSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to determine caller information")
	}

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	schemaPath := filepath.Join(repoRoot, "specs", "002-oci-helm-state", "contracts", "state-schema.json")

	compiler := jsonschema.NewCompiler()
	fh, err := os.Open(schemaPath)
	if err != nil {
		t.Fatalf("open schema: %v", err)
	}
	defer fh.Close()
	doc, err := jsonschema.UnmarshalJSON(fh)
	if err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	if err := compiler.AddResource("state-schema.json", doc); err != nil {
		t.Fatalf("add schema resource: %v", err)
	}

	schema, err := compiler.Compile("state-schema.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return schema
}

func TestStateSchemaAcceptsValidRecord(t *testing.T) {
	schema := loadStateSchema(t)
	record := map[string]any{
		"release":   "myapp-demo",
		"namespace": "demo",
		"chart": map[string]any{
			"type":      "oci",
			"reference": "oci://registry.example.com/apps/myapp:1.2.3",
			"digest":    "sha256:abc",
		},
		"version":    "1.2.3",
		"lastAction": "install",
		"timestamp":  "2025-10-05T12:34:56Z",
		"stateFile":  "state/app.json",
	}

	if err := schema.Validate(record); err != nil {
		t.Fatalf("expected record to satisfy schema, got %v", err)
	}
}

func TestStateSchemaRejectsMissingChart(t *testing.T) {
	schema := loadStateSchema(t)
	record := map[string]any{
		"release":    "myapp-demo",
		"namespace":  "demo",
		"version":    "1.2.3",
		"lastAction": "install",
		"timestamp":  "2025-10-05T12:34:56Z",
	}

	if err := schema.Validate(record); err == nil {
		t.Fatal("expected schema validation to fail when chart metadata missing")
	}
}
