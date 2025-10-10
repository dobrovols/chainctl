package loggingcontracts_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/xeipuuv/gojsonschema"
)

func TestStructuredLogSchemaAcceptsValidEntry(t *testing.T) {
	schemaLoader := gojsonschema.NewReferenceLoader(loadSchemaPath(t))
	document := map[string]any{
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
		"category":      "workflow",
		"message":       "start bootstrap",
		"severity":      "info",
		"workflowId":    "123e4567-e89b-12d3-a456-426614174000",
		"step":          "bootstrap",
		"command":       "helm upgrade myapp",
		"stderrExcerpt": "",
		"metadata": map[string]string{
			"mode":     "bootstrap",
			"duration": "1200ms",
		},
	}
	result, err := gojsonschema.Validate(schemaLoader, gojsonschema.NewGoLoader(document))
	if err != nil {
		t.Fatalf("schema validation failed: %v", err)
	}
	if !result.Valid() {
		t.Fatalf("expected document to be valid: %v", result.Errors())
	}
}

func TestStructuredLogSchemaRejectsMissingFields(t *testing.T) {
	schemaLoader := gojsonschema.NewReferenceLoader(loadSchemaPath(t))
	badDoc := map[string]any{
		"category":   "workflow",
		"message":    "missing fields",
		"severity":   "info",
		"workflowId": "123e4567-e89b-12d3-a456-426614174000",
	}
	result, err := gojsonschema.Validate(schemaLoader, gojsonschema.NewGoLoader(badDoc))
	if err != nil {
		t.Fatalf("schema validation failed: %v", err)
	}
	if result.Valid() {
		t.Fatalf("expected document to be invalid")
	}
}

func loadSchemaPath(t *testing.T) string {
	t.Helper()
	schemaPath := filepath.Join("..", "..", "specs", "003-logging", "contracts", "logging-schema.json")
	abs, err := filepath.Abs(schemaPath)
	if err != nil {
		t.Fatalf("failed to resolve schema path: %v", err)
	}
	return "file://" + abs
}
