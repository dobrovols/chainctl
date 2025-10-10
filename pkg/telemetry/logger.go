package telemetry

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"time"
)

// StructuredLogger emits structured log entries.
type StructuredLogger interface {
	Emit(Entry) error
}

// Severity represents the log severity level.
type Severity string

const (
	// SeverityInfo captures normal operation messages.
	SeverityInfo Severity = "info"
	// SeverityWarn captures recoverable anomalies.
	SeverityWarn Severity = "warn"
	// SeverityError captures unrecoverable or failure states.
	SeverityError Severity = "error"
)

// Category captures the structured log category.
type Category string

const (
	// CategoryWorkflow marks high-level workflow events.
	CategoryWorkflow Category = "workflow"
	// CategoryCommand marks external command events.
	CategoryCommand Category = "command"
	// CategoryDiagnostic marks ancillary diagnostic events.
	CategoryDiagnostic Category = "diagnostic"
)

// Entry describes a structured log entry prior to serialization.
type Entry struct {
	Category      Category
	Message       string
	Severity      Severity
	Step          string
	Command       string
	StderrExcerpt string
	Metadata      map[string]string
	Error         error
}

// Logger emits structured JSON logs.
type Logger struct {
	enc        *json.Encoder
	workflowID string
	mu         sync.Mutex
}

// NewLogger constructs a logger for a workflow.
func NewLogger(w io.Writer, workflowID string) (*Logger, error) {
	if w == nil {
		return nil, errors.New("logger writer is required")
	}
	trimmed := strings.TrimSpace(workflowID)
	if trimmed == "" {
		return nil, errors.New("workflow ID is required")
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &Logger{enc: enc, workflowID: trimmed}, nil
}

// Emit writes the provided entry to the underlying writer.
func (l *Logger) Emit(entry Entry) error {
	if l == nil {
		return errors.New("logger is nil")
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	severity := entry.Severity
	if severity == "" {
		severity = SeverityInfo
	}

	metadata := map[string]string{}
	if len(entry.Metadata) > 0 {
		metadata = make(map[string]string, len(entry.Metadata))
		for k, v := range entry.Metadata {
			metadata[k] = v
		}
	}

	if entry.Error != nil {
		severity = SeverityError
		metadata["error"] = entry.Error.Error()
	}

	payload := map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"category":   string(entry.Category),
		"message":    entry.Message,
		"severity":   string(severity),
		"workflowId": l.workflowID,
	}

	if entry.Step != "" {
		payload["step"] = entry.Step
	}
	if entry.Command != "" {
		payload["command"] = entry.Command
	}
	if entry.StderrExcerpt != "" {
		payload["stderrExcerpt"] = entry.StderrExcerpt
	}
	if len(metadata) > 0 {
		payload["metadata"] = metadata
	}

	return l.enc.Encode(payload)
}
