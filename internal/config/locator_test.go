package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/dobrovols/chainctl/internal/config"
)

func TestLocateConfigExplicitPathHasPriority(t *testing.T) {
	tmpDir := t.TempDir()
	explicitPath := filepath.Join(tmpDir, "explicit.yaml")
	mustWriteFile(t, explicitPath, "key: value")

	t.Setenv("CHAINCTL_CONFIG", filepath.Join(tmpDir, "env.yaml"))
	mustWriteFile(t, os.Getenv("CHAINCTL_CONFIG"), "env: value")

	result, err := config.LocateConfig(explicitPath)
	if err != nil {
		t.Fatalf("LocateConfig returned error: %v", err)
	}
	if result.Path != explicitPath {
		t.Fatalf("expected explicit path %q, got %q", explicitPath, result.Path)
	}
	if result.Source != config.ConfigSourceExplicit {
		t.Fatalf("expected explicit source, got %s", result.Source)
	}
}

func TestLocateConfigEnvironmentVariable(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, "env.yaml")
	mustWriteFile(t, envPath, "env: value")

	t.Setenv("CHAINCTL_CONFIG", envPath)

	result, err := config.LocateConfig("")
	if err != nil {
		t.Fatalf("LocateConfig returned error: %v", err)
	}
	if result.Path != envPath {
		t.Fatalf("expected env path %q, got %q", envPath, result.Path)
	}
	if result.Source != config.ConfigSourceEnv {
		t.Fatalf("expected env source, got %s", result.Source)
	}
}

func TestLocateConfigWorkingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	restoreWD := changeWorkingDir(t, tmpDir)
	t.Cleanup(restoreWD)

	wdPath := filepath.Join(tmpDir, "chainctl.yaml")
	mustWriteFile(t, wdPath, "wd: value")

	t.Setenv("CHAINCTL_CONFIG", "")

	result, err := config.LocateConfig("")
	if err != nil {
		t.Fatalf("LocateConfig returned error: %v", err)
	}
	expectedPath, err := filepath.EvalSymlinks(wdPath)
	if err != nil {
		t.Fatalf("eval symlinks: %v", err)
	}
	actualPath, err := filepath.EvalSymlinks(result.Path)
	if err != nil {
		t.Fatalf("eval result symlinks: %v", err)
	}
	if actualPath != expectedPath {
		t.Fatalf("expected working directory path %q, got %q", expectedPath, actualPath)
	}
	if result.Source != config.ConfigSourceWorkingDir {
		t.Fatalf("expected working-dir source, got %s", result.Source)
	}
}

func TestLocateConfigXDGDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CHAINCTL_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	xdgPath := filepath.Join(tmpDir, "chainctl", "config.yaml")
	mustWriteFile(t, xdgPath, "xdg: value")

	result, err := config.LocateConfig("")
	if err != nil {
		t.Fatalf("LocateConfig returned error: %v", err)
	}
	if result.Path != xdgPath {
		t.Fatalf("expected XDG path %q, got %q", xdgPath, result.Path)
	}
	if result.Source != config.ConfigSourceXDG {
		t.Fatalf("expected XDG source, got %s", result.Source)
	}
}

func TestLocateConfigHomeDirectoryFallback(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CHAINCTL_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", tmpDir)

	homePath := filepath.Join(tmpDir, ".config", "chainctl", "config.yaml")
	mustWriteFile(t, homePath, "home: value")

	result, err := config.LocateConfig("")
	if err != nil {
		t.Fatalf("LocateConfig returned error: %v", err)
	}
	if result.Path != homePath {
		t.Fatalf("expected home fallback path %q, got %q", homePath, result.Path)
	}
	if result.Source != config.ConfigSourceHome {
		t.Fatalf("expected home source, got %s", result.Source)
	}
}

func TestLocateConfigMissingReturnsError(t *testing.T) {
	t.Setenv("CHAINCTL_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	_, err := config.LocateConfig("")
	if err == nil {
		t.Fatalf("expected error when no configuration file is present")
	}
	if !errors.Is(err, config.ErrConfigNotFound) {
		t.Fatalf("expected ErrConfigNotFound, got %v", err)
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func changeWorkingDir(t *testing.T, dir string) func() {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	return func() {
		_ = os.Chdir(original)
	}
}
