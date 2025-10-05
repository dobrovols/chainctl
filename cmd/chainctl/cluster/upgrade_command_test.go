package cluster_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"

	clustercmd "github.com/dobrovols/chainctl/cmd/chainctl/cluster"
	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/upgrade"
)

type fakePlanner struct {
	called  bool
	profile *config.Profile
	plan    upgrade.Plan
	err     error
}

func (f *fakePlanner) PlanUpgrade(profile *config.Profile, plan upgrade.Plan) error {
	f.called = true
	f.profile = profile
	f.plan = plan
	return f.err
}

func TestClusterUpgradeCommand_TextOutput(t *testing.T) {
	planner := &fakePlanner{}
	deps := clustercmd.UpgradeDeps{Planner: planner}

	opts := clustercmd.UpgradeOptions{
		ClusterEndpoint: "https://cluster.local",
		K3sVersion:      "v1.30.2+k3s1",
		Output:          "text",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := clustercmd.RunClusterUpgradeForTest(cmd, opts, deps); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	if !planner.called {
		t.Fatalf("expected planner invocation")
	}
	if !bytes.Contains(out.Bytes(), []byte("scheduled")) {
		t.Fatalf("expected text output, got %s", out.String())
	}
}

func TestClusterUpgradeCommand_JSONOutput(t *testing.T) {
	planner := &fakePlanner{}
	deps := clustercmd.UpgradeDeps{Planner: planner}

	opts := clustercmd.UpgradeOptions{
		ClusterEndpoint:    "https://cluster.local",
		K3sVersion:         "v1.30.2+k3s1",
		ControllerManifest: "manifest.yaml",
		Output:             "json",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := clustercmd.RunClusterUpgradeForTest(cmd, opts, deps); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	if !bytes.Contains(out.Bytes(), []byte("\"status\":\"scheduled\"")) {
		t.Fatalf("expected json output, got %s", out.String())
	}
}

func TestClusterUpgradeCommand_ValidatesInputs(t *testing.T) {
	deps := clustercmd.UpgradeDeps{Planner: &fakePlanner{}}

	err := clustercmd.RunClusterUpgradeForTest(&cobra.Command{}, clustercmd.UpgradeOptions{}, deps)
	if err != clustercmd.ErrClusterEndpoint() {
		t.Fatalf("expected cluster endpoint error, got %v", err)
	}

	err = clustercmd.RunClusterUpgradeForTest(&cobra.Command{}, clustercmd.UpgradeOptions{ClusterEndpoint: "https://cluster"}, deps)
	if err != clustercmd.ErrK3sVersion() {
		t.Fatalf("expected k3s version error, got %v", err)
	}
}
