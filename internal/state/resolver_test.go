package state

import (
	"os"
	"path/filepath"
	"testing"

	pkgstate "github.com/dobrovols/chainctl/pkg/state"
)

func TestResolverUsesAbsoluteOverrides(t *testing.T) {
	r := NewResolver()
	path, err := r.Resolve(pkgstate.Overrides{StateFilePath: "/tmp/custom.json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "/tmp/custom.json" {
		t.Fatalf("expected /tmp/custom.json, got %s", path)
	}
}

func TestResolverRejectsRelativePath(t *testing.T) {
	r := NewResolver()
	if _, err := r.Resolve(pkgstate.Overrides{StateFilePath: "relative.json"}); err == nil {
		t.Fatal("expected error for relative path")
	}
}

func TestResolverCreatesFilenameWithinDirectory(t *testing.T) {
	r := NewResolver()
	dir := t.TempDir()
	path, err := r.Resolve(pkgstate.Overrides{StateDirectory: dir, StateFileName: "custom.json"})
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	expected := filepath.Join(dir, "custom.json")
	if path != expected {
		t.Fatalf("expected %s, got %s", expected, path)
	}
}

func TestResolverDefaultsToConfigDirectory(t *testing.T) {
	r := NewResolver()
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "cfg"))
	t.Cleanup(func() { os.Unsetenv("XDG_CONFIG_HOME") })

	path, err := r.Resolve(pkgstate.Overrides{})
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if filepath.Base(path) != "app.json" {
		t.Fatalf("expected default filename, got %s", filepath.Base(path))
	}
}

func TestResolverConflictingOverrides(t *testing.T) {
	r := NewResolver()
	if _, err := r.Resolve(pkgstate.Overrides{StateFilePath: "/tmp/custom.json", StateFileName: "custom.json"}); err == nil {
		t.Fatal("expected error for conflicting overrides")
	}
}

func TestResolverInvalidFileName(t *testing.T) {
	r := NewResolver()
	if _, err := r.Resolve(pkgstate.Overrides{StateDirectory: t.TempDir(), StateFileName: "subdir/state.json"}); err == nil {
		t.Fatal("expected invalid filename error")
	}
}

func TestResolverDefaultsToHomeDirectory(t *testing.T) {
	r := NewResolver()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", t.TempDir())

	path, err := r.Resolve(pkgstate.Overrides{})
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if filepath.Base(path) != "app.json" {
		t.Fatalf("expected default filename, got %s", filepath.Base(path))
	}
}

func TestResolverNormalisesRelativeDirectory(t *testing.T) {
	r := NewResolver()
	path, err := r.Resolve(pkgstate.Overrides{StateDirectory: "relative/dir", StateFileName: "state.json"})
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("expected absolute path, got %s", path)
	}
}

func TestErrorAccessorsExposeSentinels(t *testing.T) {
	if ErrConflictingOverrides() != errConflictingOverrides {
		t.Fatal("conflicting overrides accessor should expose sentinel")
	}
	if ErrRelativeStateFile() != errRelativeStateFile {
		t.Fatal("relative state file accessor should expose sentinel")
	}
	if ErrInvalidFileName() != errInvalidFileName {
		t.Fatal("invalid file name accessor should expose sentinel")
	}
}

func TestInvalidFileNameHelper(t *testing.T) {
	if !invalidFileName("") {
		t.Fatal("expected empty name to be invalid")
	}
	if !invalidFileName("nested/file.json") {
		t.Fatal("expected path separator to be invalid")
	}
	if invalidFileName("state.json") {
		t.Fatal("expected simple filename to be valid")
	}
}
