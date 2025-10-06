package e2e

import (
	"os"
	"testing"
)

func TestClusterUpgradePlan(t *testing.T) {
	if os.Getenv("CHAINCTL_E2E") == "" {
		t.Skip("skip cluster upgrade e2e: set CHAINCTL_E2E=1")
	}

	cmd := goCommand(t, projectRoot(t), []string{"GO111MODULE=on"},
		"run", "./cmd/chainctl", "cluster", "upgrade",
		"--cluster-endpoint", envOrDefault("CHAINCTL_CLUSTER_ENDPOINT", "https://cluster.local"),
		"--k3s-version", envOrDefault("CHAINCTL_K3S_VERSION", "v1.30.2+k3s1"),
		"--output", "json",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cluster upgrade dry-run failed: %v\n%s", err, string(out))
	}
}
