package telemetry_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/dobrovols/chainctl/pkg/telemetry"
)

func TestEmitterEmit(t *testing.T) {
	var buf bytes.Buffer
	emitter, err := telemetry.NewEmitter(&buf)
	if err != nil {
		t.Fatalf("new emitter: %v", err)
	}

	err = emitter.Emit(telemetry.Event{Phase: telemetry.PhasePreflight, Outcome: "start", Metadata: map[string]string{"cluster": "dev"}})
	if err != nil {
		t.Fatalf("emit: %v", err)
	}

	var ev telemetry.Event
	if err := json.NewDecoder(&buf).Decode(&ev); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ev.Phase != telemetry.PhasePreflight {
		t.Fatalf("expected phase preflight, got %s", ev.Phase)
	}
	if ev.Metadata["cluster"] != "dev" {
		t.Fatalf("metadata missing")
	}
}

func TestEmitterEmitPhasePropagatesError(t *testing.T) {
	var buf bytes.Buffer
	emitter, err := telemetry.NewEmitter(&buf)
	if err != nil {
		t.Fatalf("new emitter: %v", err)
	}

	sampleErr := errors.New("boom")
	err = emitter.EmitPhase(telemetry.PhaseHelm, map[string]string{"release": "chainapp"}, func() error {
		return sampleErr
	})
	if !errors.Is(err, sampleErr) {
		t.Fatalf("expected wrapped error, got %v", err)
	}

	dec := json.NewDecoder(&buf)
	var start telemetry.Event
	for start.Phase == "" {
		if err := dec.Decode(&start); err != nil {
			t.Fatalf("decode start: %v", err)
		}
	}
	if start.Phase != telemetry.PhaseHelm {
		t.Fatalf("expected helm phase start, got %+v", start)
	}
	var end telemetry.Event
	for end.Phase == "" {
		if err := dec.Decode(&end); err != nil {
			t.Fatalf("decode end: %v", err)
		}
	}
	if end.Outcome != "failure" {
		t.Fatalf("expected failure outcome, got %s", end.Outcome)
	}
	if end.Duration <= 0 {
		t.Fatalf("expected duration to be set")
	}
}
