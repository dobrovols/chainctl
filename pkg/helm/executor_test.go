package helm

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

type recordingExecutor struct {
	called bool
	err    error
}

func (r *recordingExecutor) UpgradeRelease(*config.Profile, *bundle.Bundle) error {
	r.called = true
	return r.err
}

func TestLoggingExecutorEmitsEntries(t *testing.T) {
	recorder := &recordingExecutor{}
	var buf bytes.Buffer
	logger, err := telemetry.NewLogger(&buf, "wf-1")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	exec := NewLoggingExecutor(recorder, logger)
	profile := &config.Profile{
		HelmRelease:   "chainapp",
		HelmNamespace: "chain-system",
		EncryptedFile: "/tmp/values.enc",
		Passphrase:    "super-secret",
		BundlePath:    "/tmp/bundle",
	}

	if err := exec.UpgradeRelease(profile, &bundle.Bundle{Manifest: bundle.Manifest{Version: "1.2.3"}}); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if !recorder.called {
		t.Fatalf("expected delegate executor to be called")
	}

	entries := decodeLogs(t, buf.Bytes())
	if len(entries) != 2 {
		t.Fatalf("expected two log entries, got %d", len(entries))
	}

	start := entries[0]
	if start["severity"] != string(telemetry.SeverityInfo) {
		t.Fatalf("expected info severity, got %v", start["severity"])
	}
	command := start["command"].(string)
	if command == "" {
		t.Fatalf("expected command to be populated")
	}
	if bytes.Contains([]byte(command), []byte("super-secret")) {
		t.Fatalf("expected passphrase to be redacted, command=%q", command)
	}

	complete := entries[1]
	if complete["severity"] != string(telemetry.SeverityInfo) {
		t.Fatalf("expected completion to remain info, got %v", complete["severity"])
	}
	metadata := complete["metadata"].(map[string]any)
	if metadata["bundleVersion"] != "1.2.3" {
		t.Fatalf("expected bundle version metadata, got %v", metadata["bundleVersion"])
	}
}

func TestLoggingExecutorEscalatesOnError(t *testing.T) {
	recorder := &recordingExecutor{err: errors.New("boom")}
	var buf bytes.Buffer
	logger, err := telemetry.NewLogger(&buf, "wf-2")
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	exec := NewLoggingExecutor(recorder, logger)
	err = exec.UpgradeRelease(&config.Profile{HelmRelease: "chainapp"}, nil)
	if !errors.Is(err, recorder.err) {
		t.Fatalf("expected original error, got %v", err)
	}

	entries := decodeLogs(t, buf.Bytes())
	if len(entries) != 2 {
		t.Fatalf("expected two entries, got %d", len(entries))
	}
	if entries[1]["severity"] != string(telemetry.SeverityError) {
		t.Fatalf("expected error severity, got %v", entries[1]["severity"])
	}
	meta := entries[1]["metadata"].(map[string]any)
	if meta["error"] != "boom" {
		t.Fatalf("expected error metadata, got %v", meta["error"])
	}
}

func decodeLogs(t *testing.T, data []byte) []map[string]any {
	t.Helper()
	var out []map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var entry map[string]any
		if err := dec.Decode(&entry); err != nil {
			t.Fatalf("decode: %v", err)
		}
		out = append(out, entry)
	}
	return out
}
