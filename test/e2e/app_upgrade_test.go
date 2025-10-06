package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAppUpgradeDryRun(t *testing.T) {
	if os.Getenv("CHAINCTL_E2E") == "" {
		t.Skip("skip app upgrade e2e: set CHAINCTL_E2E=1")
	}

	tempDir := t.TempDir()
	bundlePath := filepath.Join(tempDir, "quickstart-bundle.tar")
	createQuickstartBundle(t, bundlePath)

	statePath := filepath.Join(tempDir, "state", "app.json")

	env := append(os.Environ(),
		"GO111MODULE=on",
		"XDG_CONFIG_HOME="+filepath.Join(tempDir, "xdg"),
	)

	args := []string{
		"run", "./cmd/chainctl", "app", "upgrade",
		"--bundle-path", bundlePath,
		"--cluster-endpoint", envOrDefault("CHAINCTL_CLUSTER_ENDPOINT", "https://cluster.local"),
		"--values-file", envOrDefault("CHAINCTL_VALUES_FILE", "test/e2e/testdata/values.enc"),
		"--values-passphrase", envOrDefault("CHAINCTL_VALUES_PASSPHRASE", "secret"),
		"--release-name", "quickstart",
		"--namespace", "quickstart",
		"--state-file", statePath,
		"--output", "json",
	}

	cmd := exec.Command("go", args...)
	cmd.Dir = projectRoot(t)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("app upgrade dry-run failed: %v\n%s", err, string(out))
	}

	payload := parseCLIJSON(t, out)
	assertEqual(t, payload["status"], "success", "unexpected upgrade status")
	assertEqual(t, payload["action"], "upgrade", "unexpected upgrade action")
	assertEqual(t, payload["stateFile"], statePath, "upgrade state path mismatch")

	record := readStateRecord(t, statePath)
	if record.LastAction != "upgrade" {
		t.Fatalf("expected last action upgrade, got %s", record.LastAction)
	}
	if record.Chart.Reference != bundlePath {
		t.Fatalf("expected chart reference %s, got %s", bundlePath, record.Chart.Reference)
	}
}
