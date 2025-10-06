package e2e

import (
	"os"
	"testing"
)

func TestNodeJoinDryRun(t *testing.T) {
	if os.Getenv("CHAINCTL_E2E") == "" {
		t.Skip("skip node join e2e: set CHAINCTL_E2E=1")
	}

	token := os.Getenv("CHAINCTL_JOIN_TOKEN")
	if token == "" {
		t.Skip("CHAINCTL_JOIN_TOKEN not provided")
	}

	cmd := goCommand(t, projectRoot(t), []string{"GO111MODULE=on"},
		"run", "./cmd/chainctl", "node", "join",
		"--cluster-endpoint", envOrDefault("CHAINCTL_CLUSTER_ENDPOINT", "https://cluster.local"),
		"--role", envOrDefault("CHAINCTL_JOIN_ROLE", "worker"),
		"--token", token,
		"--output", "json",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("node join dry-run failed: %v\n%s", err, string(out))
	}
}
