package e2e

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeclarativeConfigClusterInstallDryRun(t *testing.T) {
	if os.Getenv("CHAINCTL_E2E") == "" {
		t.Skip("skip declarative config e2e: set CHAINCTL_E2E=1 and provide KIND_KUBECONFIG")
	}
	if os.Getenv("CHAINCTL_E2E_SUDO") != "1" {
		t.Skip("skip declarative config e2e: set CHAINCTL_E2E_SUDO=1 to permit sudo execution")
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "chainctl.yaml")

	valuesFile := envOrDefault("CHAINCTL_VALUES_FILE", "test/e2e/testdata/values.enc")
	namespace := envOrDefault("CHAINCTL_NAMESPACE", "demo")

	configBody := []byte(`defaults:
  namespace: ` + namespace + `
  values-file: "` + valuesFile + `"
commands:
  chainctl cluster install:
    flags:
      bootstrap: true
      dry-run: true
      output: json
`)

	if err := os.WriteFile(configPath, configBody, 0o600); err != nil {
		t.Fatalf("write declarative config: %v", err)
	}

	args := []string{
		"run", "./cmd/chainctl", "cluster", "install",
		"--config", configPath,
		"--values-passphrase", envOrDefault("CHAINCTL_VALUES_PASSPHRASE", "secret"),
	}

	cmd := goCommand(t, projectRoot(t), []string{"GO111MODULE=on"}, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("declarative config cluster install dry-run failed: %v\n%s", err, string(out))
	}
}
