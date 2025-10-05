package app_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"

	appcmd "github.com/dobrovols/chainctl/cmd/chainctl/app"
	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

type fakeHelmInstaller struct {
	called bool
	err    error
}

func (f *fakeHelmInstaller) Install(p *config.Profile, b *bundle.Bundle) error {
	f.called = true
	return f.err
}

func telemetryNoop(w io.Writer) *telemetry.Emitter {
	return telemetry.NewEmitter(w)
}

func TestNewAppCommandRegistersUpgrade(t *testing.T) {
	cmd := appcmd.NewAppCommand()
	if cmd.Use != "app" {
		t.Fatalf("expected use app, got %s", cmd.Use)
	}
	var found bool
	for _, sub := range cmd.Commands() {
		if sub.Name() == "upgrade" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected upgrade subcommand to be registered")
	}
}

func TestNewUpgradeCommandFlags(t *testing.T) {
	cmd := appcmd.NewUpgradeCommand()
	for _, name := range []string{"cluster-endpoint", "values-file", "values-passphrase", "bundle-path", "airgapped", "output"} {
		if cmd.Flag(name) == nil {
			t.Fatalf("expected flag %s to exist", name)
		}
	}
}

func TestAppUpgradeCommand_TextSuccess(t *testing.T) {
	installer := &fakeHelmInstaller{}
	deps := appcmd.UpgradeDeps{
		Installer:        installer,
		TelemetryEmitter: telemetryNoop,
	}

	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		Output:           "text",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := appcmd.RunUpgradeForTest(cmd, opts, deps); err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}

	if !installer.called {
		t.Fatalf("expected installer to be called")
	}
	if !bytes.Contains(out.Bytes(), []byte("Upgrade completed")) {
		t.Fatalf("expected success message, got %s", out.String())
	}
}

func TestAppUpgradeCommand_JSONOutput(t *testing.T) {
	installer := &fakeHelmInstaller{}
	deps := appcmd.UpgradeDeps{
		Installer:        installer,
		TelemetryEmitter: telemetryNoop,
	}

	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		Output:           "json",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := appcmd.RunUpgradeForTest(cmd, opts, deps); err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}

	if !bytes.Contains(out.Bytes(), []byte("\"status\":\"success\"")) {
		t.Fatalf("expected json output, got %s", out.String())
	}
}

func TestAppUpgradeCommand_ValidatesInputs(t *testing.T) {
	deps := appcmd.UpgradeDeps{Installer: &fakeHelmInstaller{}}

	err := appcmd.RunUpgradeForTest(&cobra.Command{}, appcmd.UpgradeOptions{}, deps)
	if err != appcmd.ErrValuesFileRequired() {
		t.Fatalf("expected values file error, got %v", err)
	}

	opts := appcmd.UpgradeOptions{ValuesFile: "/tmp/values.enc", ValuesPassphrase: "secret"}
	err = appcmd.RunUpgradeForTest(&cobra.Command{}, opts, deps)
	if err != appcmd.ErrClusterEndpointRequired() {
		t.Fatalf("expected cluster endpoint error, got %v", err)
	}
}

func TestAppUpgradeCommand_UnsupportedOutput(t *testing.T) {
	deps := appcmd.UpgradeDeps{Installer: &fakeHelmInstaller{}, TelemetryEmitter: telemetryNoop}
	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		Output:           "yaml",
	}

	err := appcmd.RunUpgradeForTest(&cobra.Command{}, opts, deps)
	if !errors.Is(err, appcmd.ErrUnsupportedOutput()) {
		t.Fatalf("expected unsupported output error, got %v", err)
	}
}

func TestAppUpgradeCommand_AirgappedLoadsBundle(t *testing.T) {
	var called bool
	deps := appcmd.UpgradeDeps{
		Installer:        &fakeHelmInstaller{},
		TelemetryEmitter: telemetryNoop,
		BundleLoader: func(path, cache string) (*bundle.Bundle, error) {
			called = true
			if path != "/mnt/package.tar" {
				t.Fatalf("unexpected path %s", path)
			}
			if cache == "" {
				t.Fatalf("expected cache directory")
			}
			return &bundle.Bundle{}, nil
		},
	}

	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		BundlePath:       "/mnt/package.tar",
		Airgapped:        true,
		Output:           "text",
	}

	if err := appcmd.RunUpgradeForTest(&cobra.Command{}, opts, deps); err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}
	if !called {
		t.Fatalf("expected bundle loader to be called")
	}
}
