package appintegration_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/cobra"

	appcmd "github.com/dobrovols/chainctl/cmd/chainctl/app"
	"github.com/dobrovols/chainctl/pkg/helm"
	"github.com/dobrovols/chainctl/pkg/state"
)

type updateResolver struct {
	result helm.ResolveResult
	called bool
}

func (u *updateResolver) Resolve(ctx context.Context, opts helm.ResolveOptions) (helm.ResolveResult, error) {
	u.called = true
	return u.result, nil
}

type captureState struct {
	record state.Record
	path   string
	called bool
}

func (c *captureState) Write(rec state.Record, overrides state.Overrides) (string, error) {
	c.called = true
	c.record = rec
	c.path = overrides.StateFilePath
	return overrides.StateFilePath, nil
}

func TestUpgradeWithOCIReferenceUpdatesVersion(t *testing.T) {
	resolver := &updateResolver{result: helm.ResolveResult{
		Source:    state.ChartSource{Type: "oci", Reference: "oci://registry.example.com/apps/myapp:2.0.0", Digest: "sha256:def"},
		ChartPath: "/tmp/chart",
	}}
	stateCapture := &captureState{}
	deps := appcmd.UpgradeDeps{
		Installer:        noopInstaller{},
		TelemetryEmitter: telemetrySilentEmitter,
		Resolver:         resolver,
		StateManager:     stateCapture,
	}

	stateFilePath := t.TempDir() + "/state.json"
	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		ChartReference:   "oci://registry.example.com/apps/myapp:2.0.0",
		ReleaseName:      "myapp-demo",
		Namespace:        "demo",
		AppVersion:       "2.0.0",
		StateFilePath:    stateFilePath,
		Output:           "json",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := appcmd.RunUpgradeForTest(cmd, opts, deps); err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}

	if !resolver.called {
		t.Fatalf("expected resolver to be invoked")
	}
	if !stateCapture.called {
		t.Fatalf("expected state manager to persist record")
	}
	if stateCapture.record.LastAction != "upgrade" {
		t.Fatalf("expected last action upgrade, got %s", stateCapture.record.LastAction)
	}
	if stateCapture.record.Version != "2.0.0" {
		t.Fatalf("expected version 2.0.0, got %s", stateCapture.record.Version)
	}
}
