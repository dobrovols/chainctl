package e2e

import (
	"os"
	"os/exec"
	"testing"
)

func TestClusterInstallDryRun(t *testing.T) {
	if os.Getenv("CHAINCTL_E2E") == "" {
		t.Skip("skip cluster install e2e: set CHAINCTL_E2E=1 and provide KIND_KUBECONFIG")
	}

	run := exec.Command("go", "run", "./cmd/chainctl", "cluster", "install",
		"--cluster-endpoint", os.Getenv("CHAINCTL_CLUSTER_ENDPOINT"),
		"--values-file", envOrDefault("CHAINCTL_VALUES_FILE", "testdata/e2e/values.enc"),
		"--values-passphrase", envOrDefault("CHAINCTL_VALUES_PASSPHRASE", "secret"),
		"--dry-run",
	)
	run.Dir = projectRoot(t)
	run.Env = append(os.Environ(), "GO111MODULE=on")
	if out, err := run.CombinedOutput(); err != nil {
		t.Fatalf("cluster install dry-run failed: %v\n%s", err, string(out))
	}
}
