package e2e

import (
	"os"
	"os/exec"
	"testing"
)

func TestClusterUpgradePlan(t *testing.T) {
	if os.Getenv("CHAINCTL_E2E") == "" {
		t.Skip("skip cluster upgrade e2e: set CHAINCTL_E2E=1")
	}

	cmd := exec.Command("go", "run", "./cmd/chainctl", "cluster", "upgrade",
		"--cluster-endpoint", envOrDefault("CHAINCTL_CLUSTER_ENDPOINT", "https://cluster.local"),
		"--k3s-version", envOrDefault("CHAINCTL_K3S_VERSION", "v1.30.2+k3s1"),
		"--output", "json",
	)
	cmd.Dir = projectRoot(t)
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cluster upgrade dry-run failed: %v\n%s", err, string(out))
	}
}
