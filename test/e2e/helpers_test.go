package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func projectRoot(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(cwd, "..", ".."))
}

func goCommand(t *testing.T, dir string, extraEnv []string, args ...string) *exec.Cmd {
	t.Helper()
	baseEnv := append(os.Environ(), extraEnv...)

	if os.Getenv("CHAINCTL_E2E_SUDO") == "1" {
		sudoArgs := append([]string{"-E", "go"}, args...)
		cmd := exec.Command("sudo", sudoArgs...)
		cmd.Dir = dir
		cmd.Env = baseEnv
		return cmd
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = baseEnv
	return cmd
}
