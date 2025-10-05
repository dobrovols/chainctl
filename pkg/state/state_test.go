package state_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dobrovols/chainctl/pkg/state"
)

type stubResolver struct {
	baseDir string
	last    state.Overrides
	fail    error
}

func (s *stubResolver) Resolve(overrides state.Overrides) (string, error) {
	s.last = overrides
	if s.fail != nil {
		return "", s.fail
	}
	if overrides.StateFilePath != "" {
		return overrides.StateFilePath, nil
	}
	dir := overrides.StateDirectory
	if dir == "" {
		dir = s.baseDir
	}
	name := overrides.StateFileName
	if name == "" {
		name = "app.json"
	}
	return filepath.Join(dir, name), nil
}

func sampleRecord(version string) state.Record {
	return state.Record{
		Release:   "myapp",
		Namespace: "demo",
		Chart: state.ChartSource{
			Type:      "oci",
			Reference: "oci://registry.example.com/apps/myapp:1.2.3",
			Digest:    "sha256:abc",
		},
		Version:         version,
		LastAction:      "install",
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		ClusterEndpoint: "https://127.0.0.1:6443",
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(bytes, &payload); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	return payload
}

func TestManagerCreatesStateFileWithDefaultDirectory(t *testing.T) {
	base := t.TempDir()
	stateDir := filepath.Join(base, "state")
	resolver := &stubResolver{baseDir: stateDir}
	manager := state.NewManager(resolver)
	record := sampleRecord("1.2.3")

	path, err := manager.Write(record, state.Overrides{StateDirectory: stateDir})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if path != filepath.Join(stateDir, "app.json") {
		t.Fatalf("unexpected path: %s", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if perms := info.Mode().Perm(); perms != 0o600 {
		t.Fatalf("expected file perms 0600, got %o", perms)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perms := dirInfo.Mode().Perm(); perms != 0o700 {
		t.Fatalf("expected dir perms 0700, got %o", perms)
	}

	payload := readJSON(t, path)
	if payload["version"] != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %v", payload["version"])
	}
}

func TestManagerHonorsStateFileNameOverride(t *testing.T) {
	base := t.TempDir()
	resolver := &stubResolver{baseDir: base}
	manager := state.NewManager(resolver)

	path, err := manager.Write(sampleRecord("1.2.3"), state.Overrides{StateDirectory: base, StateFileName: "custom.json"})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	expected := filepath.Join(base, "custom.json")
	if path != expected {
		t.Fatalf("expected path %s, got %s", expected, path)
	}
}

func TestManagerHonorsAbsoluteStateFilePath(t *testing.T) {
	base := t.TempDir()
	absPath := filepath.Join(base, "nested", "state.json")
	resolver := &stubResolver{baseDir: base}
	manager := state.NewManager(resolver)

	path, err := manager.Write(sampleRecord("1.2.3"), state.Overrides{StateFilePath: absPath})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if path != absPath {
		t.Fatalf("expected path %s, got %s", absPath, path)
	}
}

func TestManagerRewritesStateAtomically(t *testing.T) {
	base := t.TempDir()
	resolver := &stubResolver{baseDir: base}
	manager := state.NewManager(resolver)

	path, err := manager.Write(sampleRecord("1.2.3"), state.Overrides{StateDirectory: base})
	if err != nil {
		t.Fatalf("initial write failed: %v", err)
	}

	_, err = manager.Write(sampleRecord("2.0.0"), state.Overrides{StateDirectory: base})
	if err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	payload := readJSON(t, path)
	if payload["version"] != "2.0.0" {
		t.Fatalf("expected version 2.0.0, got %v", payload["version"])
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one entry after atomic write, got %d", len(entries))
	}
}

func TestManagerReturnsErrorWhenDirectoryReadOnly(t *testing.T) {
	base := t.TempDir()
	resolver := &stubResolver{baseDir: base}
	manager := state.NewManager(resolver)

	if err := os.Chmod(base, 0o500); err != nil {
		t.Fatalf("chmod base dir: %v", err)
	}

	_, err := manager.Write(sampleRecord("1.2.3"), state.Overrides{StateDirectory: base})
	if err == nil {
		t.Fatal("expected error when writing to read-only directory")
	}
	if !errors.Is(err, os.ErrPermission) && !os.IsPermission(err) {
		t.Fatalf("expected permission error, got %v", err)
	}
}
