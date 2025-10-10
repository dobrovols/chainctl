package telemetry

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"
)

func TestLoggerEmitPopulatesRequiredFields(t *testing.T) {
	var buf bytes.Buffer
	logger, err := NewLogger(&buf, "wf-123")
	if err != nil {
		t.Fatalf("unexpected error constructing logger: %v", err)
	}

	err = logger.Emit(Entry{
		Category: CategoryWorkflow,
		Severity: SeverityInfo,
		Message:  "starting bootstrap",
		Step:     "bootstrap",
	})
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}

	required := []string{"timestamp", "category", "message", "severity"}
	for _, key := range required {
		if _, ok := payload[key]; !ok {
			t.Fatalf("expected key %q in payload: %v", key, payload)
		}
	}

	if payload["category"] != string(CategoryWorkflow) {
		t.Fatalf("expected category %q, got %v", CategoryWorkflow, payload["category"])
	}

	if payload["workflowId"] != "wf-123" {
		t.Fatalf("expected workflowId to be propagated, got %v", payload["workflowId"])
	}

	if payload["step"] != "bootstrap" {
		t.Fatalf("expected step to be preserved, got %v", payload["step"])
	}
}

func TestLoggerEmitEscalatesSeverityOnError(t *testing.T) {
	var buf bytes.Buffer
	logger, err := NewLogger(&buf, "wf-123")
	if err != nil {
		t.Fatalf("unexpected error constructing logger: %v", err)
	}

	err = logger.Emit(Entry{
		Category:      CategoryCommand,
		Message:       "helm upgrade",
		Severity:      SeverityInfo,
		Command:       "helm upgrade myapp",
		Error:         errors.New("boom"),
		StderrExcerpt: "line 1: unauthorized",
	})
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}

	if payload["severity"] != string(SeverityError) {
		t.Fatalf("expected severity escalated to error, got %v", payload["severity"])
	}

	metadata, ok := payload["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata map, got %T", payload["metadata"])
	}

	if metadata["error"] != "boom" {
		t.Fatalf("expected error metadata to be captured, got %v", metadata["error"])
	}

	if payload["stderrExcerpt"] != "line 1: unauthorized" {
		t.Fatalf("expected stderr excerpt preserved, got %v", payload["stderrExcerpt"])
	}
}

func TestLoggerRequiresWorkflowID(t *testing.T) {
	_, err := NewLogger(io.Discard, "")
	if err == nil {
		t.Fatalf("expected error when workflow ID missing")
	}
}
