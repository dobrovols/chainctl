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

type stubResolver struct {
	opts   helm.ResolveOptions
	result helm.ResolveResult
	err    error
	called bool
}

func (s *stubResolver) Resolve(ctx context.Context, opts helm.ResolveOptions) (helm.ResolveResult, error) {
	s.called = true
	s.opts = opts
	if s.err != nil {
		return helm.ResolveResult{}, s.err
	}
	return s.result, nil
}

type stubStateManager struct {
	record    state.Record
	overrides state.Overrides
	path      string
	err       error
	called    bool
}

func (s *stubStateManager) Write(rec state.Record, overrides state.Overrides) (string, error) {
	s.called = true
	s.record = rec
	s.overrides = overrides
	if s.err != nil {
		return "", s.err
	}
	return s.path, nil
}

func TestUpgradeWithOCIReferencePersistsState(t *testing.T) {
	resolver := &stubResolver{result: helm.ResolveResult{
		Source:    state.ChartSource{Type: "oci", Reference: "oci://registry.example.com/apps/myapp:1.2.3", Digest: "sha256:abc"},
		ChartPath: "/tmp/chart",
	}}
	tmpDir := t.TempDir()
	stateFilePath := tmpDir + "/state.json"
	stateMgr := &stubStateManager{path: stateFilePath}
	deps := appcmd.UpgradeDeps{
		Installer:        noopInstaller{},
		TelemetryEmitter: telemetrySilentEmitter,
		Resolver:         resolver,
		StateManager:     stateMgr,
	}

	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		ChartReference:   "oci://registry.example.com/apps/myapp:1.2.3",
		ReleaseName:      "myapp-demo",
		Namespace:        "demo",
		AppVersion:       "1.2.3",
		StateFilePath:    stateFilePath,
		Output:           "text",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := appcmd.RunUpgradeForTest(cmd, opts, deps); err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}

	if !resolver.called {
		t.Fatalf("expected resolver to be called")
	}
	if resolver.opts.OCIReference != opts.ChartReference {
		t.Fatalf("expected resolver to receive %s, got %s", opts.ChartReference, resolver.opts.OCIReference)
	}
	if !stateMgr.called {
		t.Fatalf("expected state manager to be called")
	}
	if stateMgr.record.LastAction != "upgrade" {
		t.Fatalf("expected last action upgrade, got %s", stateMgr.record.LastAction)
	}
	if stateMgr.record.Chart.Reference != opts.ChartReference {
		t.Fatalf("expected chart reference recorded, got %s", stateMgr.record.Chart.Reference)
	}
	if stateMgr.record.Version != opts.AppVersion {
		t.Fatalf("expected version recorded, got %s", stateMgr.record.Version)
	}
	if stateMgr.overrides.StateFilePath != opts.StateFilePath {
		t.Fatalf("expected overrides to propagate state file path, got %s", stateMgr.overrides.StateFilePath)
	}
}
