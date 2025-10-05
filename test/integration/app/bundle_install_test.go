package appintegration_test

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"

	appcmd "github.com/dobrovols/chainctl/cmd/chainctl/app"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/state"
)

type capturingBundleLoader struct {
	path      string
	cacheRoot string
	called    bool
}

func (c *capturingBundleLoader) Load(path, cacheRoot string) (*bundle.Bundle, error) {
	c.called = true
	c.path = path
	c.cacheRoot = cacheRoot
	return &bundle.Bundle{}, nil
}

type bundleStateManager struct {
	record state.Record
	called bool
}

func (b *bundleStateManager) Write(rec state.Record, overrides state.Overrides) (string, error) {
	b.called = true
	b.record = rec
	return overrides.StateFilePath, nil
}

func TestUpgradeWithBundlePersistsState(t *testing.T) {
	loader := &capturingBundleLoader{}
	stateMgr := &bundleStateManager{}
	deps := appcmd.UpgradeDeps{
		Installer:        noopInstaller{},
		BundleLoader:     loader.Load,
		TelemetryEmitter: telemetrySilentEmitter,
		StateManager:     stateMgr,
	}

	stateFilePath := t.TempDir() + "/state.json"
	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		BundlePath:       "/tmp/bundle.tar",
		Airgapped:        true,
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

	if !loader.called {
		t.Fatalf("expected bundle loader to be invoked")
	}
	if loader.path != opts.BundlePath {
		t.Fatalf("expected loader path %s, got %s", opts.BundlePath, loader.path)
	}
	if !stateMgr.called {
		t.Fatalf("expected state manager to record state")
	}
	if stateMgr.record.Chart.Type != "bundle" {
		t.Fatalf("expected chart type bundle, got %s", stateMgr.record.Chart.Type)
	}
}
