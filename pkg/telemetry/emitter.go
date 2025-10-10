package telemetry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Phase represents a lifecycle step of the installer.
type Phase string

const (
	PhasePreflight Phase = "preflight"
	PhaseBootstrap Phase = "bootstrap"
	PhaseHelm      Phase = "helm"
	PhaseUpgrade   Phase = "upgrade"
	PhaseJoin      Phase = "join"
	PhaseVerify    Phase = "verify"
)

// Event captures structured telemetry emitted by the CLI.
type Event struct {
	Timestamp  time.Time         `json:"timestamp"`
	Phase      Phase             `json:"phase"`
	Outcome    string            `json:"outcome"`
	Duration   time.Duration     `json:"duration,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	WorkflowID string            `json:"workflowId,omitempty"`
}

// Emitter handles emitting JSON structured events to an io.Writer.
type Emitter struct {
	mu         sync.Mutex
	encoder    *json.Encoder
	logger     StructuredLogger
	workflowID string
}

// NewEmitter constructs an emitter writing JSON lines to w.
func NewEmitter(w io.Writer) (*Emitter, error) {
	if w == nil {
		return nil, errors.New("telemetry: writer is required")
	}
	workflowID := uuid.NewString()
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	logger, err := NewLogger(w, workflowID)
	if err != nil {
		return nil, fmt.Errorf("telemetry: create structured logger: %w", err)
	}
	return &Emitter{encoder: enc, logger: logger, workflowID: workflowID}, nil
}

// Emit writes an event to the underlying writer.
func (e *Emitter) Emit(ev Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	if ev.Metadata == nil {
		ev.Metadata = map[string]string{}
	}
	ev.WorkflowID = e.workflowID
	return e.encoder.Encode(ev)
}

// EmitPhase publishes start and completion events while executing fn.
func (e *Emitter) EmitPhase(phase Phase, metadata map[string]string, fn func() error) error {
	metaCopy := copyMetadata(metadata)
	if e.logger != nil {
		_ = e.logger.Emit(Entry{
			Category: CategoryWorkflow,
			Message:  fmt.Sprintf("%s phase started", phase),
			Severity: SeverityInfo,
			Step:     string(phase),
			Metadata: metaCopy,
		})
	}

	start := time.Now()
	if err := e.Emit(Event{Phase: phase, Outcome: "start", Metadata: metadata}); err != nil {
		return fmt.Errorf("emit start event: %w", err)
	}

	err := fn()
	outcome := "success"

	if err != nil {
		outcome = "failure"
	}

	emitErr := e.Emit(Event{Phase: phase, Outcome: outcome, Duration: time.Since(start), Metadata: metadata})
	if e.logger != nil {
		logMetadata := copyMetadata(metadata)
		logMetadata["duration"] = time.Since(start).String()
		severity := SeverityInfo
		if err != nil {
			severity = SeverityError
		}
		_ = e.logger.Emit(Entry{
			Category: CategoryWorkflow,
			Message:  fmt.Sprintf("%s phase %s", phase, outcome),
			Severity: severity,
			Step:     string(phase),
			Metadata: logMetadata,
			Error:    err,
		})
	}
	if emitErr != nil {
		return fmt.Errorf("emit completion event: %w", emitErr)
	}

	return err
}

// StructuredLogger exposes the structured logger associated with the emitter.
func (e *Emitter) StructuredLogger() StructuredLogger {
	return e.logger
}

// WorkflowID returns the unique identifier associated with the emitter session.
func (e *Emitter) WorkflowID() string {
	return e.workflowID
}

func copyMetadata(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	copy := make(map[string]string, len(src))
	for k, v := range src {
		copy[k] = v
	}
	return copy
}
