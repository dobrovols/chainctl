package appintegration_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	appcmd "github.com/dobrovols/chainctl/cmd/chainctl/app"
	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/helm"
	"github.com/dobrovols/chainctl/pkg/state"
)

type permissionStateManager struct {
	called bool
	err    error
}

func (p *permissionStateManager) Write(rec state.Record, overrides state.Overrides) (string, error) {
	p.called = true
	return "", p.err
}

type recordingInstaller struct{ called bool }

func (r *recordingInstaller) Install(*config.Profile, *bundle.Bundle) error {
	r.called = true
	return nil
}

type noopChartResolver struct{}

func (noopChartResolver) Resolve(ctx context.Context, opts helm.ResolveOptions) (helm.ResolveResult, error) {
	return helm.ResolveResult{Source: state.ChartSource{Type: "oci", Reference: opts.OCIReference}}, nil
}

func TestUpgradeStateWriteFailureSurfaceError(t *testing.T) {
	permErr := errors.New("permission denied")
	stateMgr := &permissionStateManager{err: permErr}
	installer := &recordingInstaller{}
	deps := appcmd.UpgradeDeps{
		Installer:        installer,
		TelemetryEmitter: telemetrySilentEmitter,
		Resolver:         noopChartResolver{},
		StateManager:     stateMgr,
	}

	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		ChartReference:   "oci://registry.example.com/apps/myapp:1.2.3",
		StateFilePath:    "/var/lib/chainctl/state.json",
		Output:           "text",
	}

	err := appcmd.RunUpgradeForTest(&cobra.Command{}, opts, deps)
	if err == nil {
		t.Fatalf("expected error when state manager fails")
	}
	if !stateMgr.called {
		t.Fatalf("expected state manager to be invoked")
	}
	if !installer.called {
		t.Fatalf("expected installer to run even when state persistence fails")
	}
	expected := "state file could not be written"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error message to contain %q, got %v", expected, err)
	}
}
