package e2e

import (
	"os"
	"os/exec"
	"testing"
)

func TestAppUpgradeDryRun(t *testing.T) {
	if os.Getenv("CHAINCTL_E2E") == "" {
		t.Skip("skip app upgrade e2e: set CHAINCTL_E2E=1")
	}

	args := []string{
		"run", "./cmd/chainctl", "app", "upgrade",
		"--cluster-endpoint", envOrDefault("CHAINCTL_CLUSTER_ENDPOINT", "https://cluster.local"),
		"--values-file", envOrDefault("CHAINCTL_VALUES_FILE", "testdata/e2e/values.enc"),
		"--values-passphrase", envOrDefault("CHAINCTL_VALUES_PASSPHRASE", "secret"),
		"--output", "json",
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = projectRoot(t)
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("app upgrade dry-run failed: %v\n%s", err, string(out))
	}
}
