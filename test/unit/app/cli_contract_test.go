package appcontracts_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	appcmd "github.com/dobrovols/chainctl/cmd/chainctl/app"
	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/helm"
	pkgstate "github.com/dobrovols/chainctl/pkg/state"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

type contractInstaller struct{}

func (contractInstaller) Install(*config.Profile, *bundle.Bundle) error { return nil }

type contractResolver struct {
	result helm.ResolveResult
}

func (c contractResolver) Resolve(ctx context.Context, opts helm.ResolveOptions) (helm.ResolveResult, error) {
	return c.result, nil
}

type contractStateManager struct {
	path string
	err  error
}

func (c contractStateManager) Write(rec pkgstate.Record, overrides pkgstate.Overrides) (string, error) {
	if c.err != nil {
		return "", c.err
	}
	if overrides.StateFilePath != "" {
		return overrides.StateFilePath, nil
	}
	return c.path, nil
}

func silentTelemetryEmitter(io.Writer) *telemetry.Emitter {
	return telemetry.NewEmitter(io.Discard)
}

func TestUpgradeCommandTextOutputContract(t *testing.T) {
	deps := appcmd.UpgradeDeps{
		Installer:        contractInstaller{},
		TelemetryEmitter: silentTelemetryEmitter,
		Resolver: contractResolver{result: helm.ResolveResult{Source: pkgstate.ChartSource{
			Type:      "oci",
			Reference: "oci://registry.example.com/apps/myapp:1.2.3",
		}}},
		StateManager: contractStateManager{path: "/Users/alex/.config/chainctl/state/app.json"},
	}

	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		ChartReference:   "oci://registry.example.com/apps/myapp:1.2.3",
		ReleaseName:      "myapp-demo",
		Namespace:        "demo",
		StateFilePath:    "/Users/alex/.config/chainctl/state/app.json",
		Output:           "text",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := appcmd.RunUpgradeForTest(cmd, opts, deps); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	expected := "Upgrade completed successfully for release myapp-demo in namespace demo\n" +
		"State written to /Users/alex/.config/chainctl/state/app.json\n"
	if out.String() != expected {
		t.Fatalf("text contract mismatch. expected:\n%s\nactual:\n%s", expected, out.String())
	}
}

func TestUpgradeCommandJSONOutputContract(t *testing.T) {
	deps := appcmd.UpgradeDeps{
		Installer:        contractInstaller{},
		TelemetryEmitter: silentTelemetryEmitter,
		Resolver: contractResolver{result: helm.ResolveResult{Source: pkgstate.ChartSource{
			Type:      "oci",
			Reference: "oci://registry.example.com/apps/myapp:1.2.3",
		}}},
		StateManager: contractStateManager{path: "/Users/alex/.config/chainctl/state/app.json"},
	}

	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		ChartReference:   "oci://registry.example.com/apps/myapp:1.2.3",
		ReleaseName:      "myapp-demo",
		Namespace:        "demo",
		StateFilePath:    "/Users/alex/.config/chainctl/state/app.json",
		Output:           "json",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := appcmd.RunUpgradeForTest(cmd, opts, deps); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &payload); err != nil {
		t.Fatalf("failed to decode json output: %v", err)
	}

	expectations := map[string]any{
		"status":    "success",
		"action":    "upgrade",
		"release":   "myapp-demo",
		"namespace": "demo",
		"chart":     "oci://registry.example.com/apps/myapp:1.2.3",
		"stateFile": "/Users/alex/.config/chainctl/state/app.json",
	}

	for key, expected := range expectations {
		if payload[key] != expected {
			t.Fatalf("expected %s to be %v, got %v", key, expected, payload[key])
		}
	}
}

func TestUpgradeCommandInvalidStateOverrideContract(t *testing.T) {
	deps := appcmd.UpgradeDeps{
		Installer: contractInstaller{},
		Resolver:  contractResolver{result: helm.ResolveResult{Source: pkgstate.ChartSource{Type: "oci", Reference: "oci://registry.example.com/apps/myapp:1.2.3"}}},
	}
	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		ChartReference:   "oci://registry.example.com/apps/myapp:1.2.3",
		StateFilePath:    "/tmp/state.json",
		StateFileName:    "custom.json",
	}

	err := appcmd.RunUpgradeForTest(&cobra.Command{}, opts, deps)
	if err == nil {
		t.Fatalf("expected error for conflicting state overrides")
	}
	expected := "state file override is invalid"
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error containing %q, got %v", expected, err)
	}
}
