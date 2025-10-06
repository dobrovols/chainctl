package e2e

import (
	"os"
	"testing"
)

func TestClusterInstallDryRun(t *testing.T) {
	if os.Getenv("CHAINCTL_E2E") == "" {
		t.Skip("skip cluster install e2e: set CHAINCTL_E2E=1 and provide KIND_KUBECONFIG")
	}

	cmd := goCommand(t, projectRoot(t), []string{"GO111MODULE=on"},
		"run", "./cmd/chainctl", "cluster", "install",
		"--cluster-endpoint", os.Getenv("CHAINCTL_CLUSTER_ENDPOINT"),
		"--values-file", envOrDefault("CHAINCTL_VALUES_FILE", "test/e2e/testdata/values.enc"),
		"--values-passphrase", envOrDefault("CHAINCTL_VALUES_PASSPHRASE", "secret"),
		"--dry-run",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cluster install dry-run failed: %v\n%s", err, string(out))
	}
}
