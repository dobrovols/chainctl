package telemetry

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"
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
	Timestamp time.Time         `json:"timestamp"`
	Phase     Phase             `json:"phase"`
	Outcome   string            `json:"outcome"`
	Duration  time.Duration     `json:"duration,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Emitter handles emitting JSON structured events to an io.Writer.
type Emitter struct {
	mu      sync.Mutex
	encoder *json.Encoder
}

// NewEmitter constructs an emitter writing JSON lines to w.
func NewEmitter(w io.Writer) *Emitter {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &Emitter{encoder: enc}
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
	return e.encoder.Encode(ev)
}

// EmitPhase publishes start and completion events while executing fn.
func (e *Emitter) EmitPhase(phase Phase, metadata map[string]string, fn func() error) error {
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
	if emitErr != nil {
		return fmt.Errorf("emit completion event: %w", emitErr)
	}

	return err
}
