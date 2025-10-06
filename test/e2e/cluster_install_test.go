package e2e

import (
	"os"
	"testing"
)

func TestClusterInstallDryRun(t *testing.T) {
	if os.Getenv("CHAINCTL_E2E") == "" {
		t.Skip("skip cluster install e2e: set CHAINCTL_E2E=1 and provide KIND_KUBECONFIG")
	}
	if os.Getenv("CHAINCTL_E2E_SUDO") != "1" {
		t.Skip("skip cluster install e2e: set CHAINCTL_E2E_SUDO=1 to permit sudo execution")
	}

	args := []string{
		"run", "./cmd/chainctl", "cluster", "install",
		"--values-file", envOrDefault("CHAINCTL_VALUES_FILE", "test/e2e/testdata/values.enc"),
		"--values-passphrase", envOrDefault("CHAINCTL_VALUES_PASSPHRASE", "secret"),
		"--dry-run",
	}

	if os.Getenv("CHAINCTL_E2E_CLUSTER_REUSE") == "1" {
		endpoint := os.Getenv("CHAINCTL_CLUSTER_ENDPOINT")
		if endpoint == "" {
			t.Skip("CHAINCTL_E2E_CLUSTER_REUSE=1 requires CHAINCTL_CLUSTER_ENDPOINT")
		}
		args = append(args, "--cluster-endpoint", endpoint)
	} else {
		args = append(args, "--bootstrap")
	}

	cmd := goCommand(t, projectRoot(t), []string{"GO111MODULE=on"}, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cluster install dry-run failed: %v\n%s", err, string(out))
	}
}
