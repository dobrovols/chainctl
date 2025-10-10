package app_test

import (
	"bytes"
	"context"
	"errors"
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

type fakeHelmInstaller struct {
	called bool
	err    error
}

func (f *fakeHelmInstaller) Install(p *config.Profile, b *bundle.Bundle) error {
	f.called = true
	return f.err
}

type resolvingStub struct {
	result helm.ResolveResult
	err    error
	called bool
}

func (r *resolvingStub) Resolve(ctx context.Context, opts helm.ResolveOptions) (helm.ResolveResult, error) {
	r.called = true
	return r.result, r.err
}

type stateStub struct {
	path   string
	err    error
	record pkgstate.Record
	called bool
}

func (s *stateStub) Write(rec pkgstate.Record, o pkgstate.Overrides) (string, error) {
	s.called = true
	s.record = rec
	if s.err != nil {
		return "", s.err
	}
	if o.StateFilePath != "" {
		s.path = o.StateFilePath
	}
	return s.path, nil
}

func telemetryNoop(w io.Writer) (*telemetry.Emitter, error) {
	return telemetry.NewEmitter(w)
}

func TestNewAppCommandRegistersSubcommands(t *testing.T) {
	cmd := appcmd.NewAppCommand()
	if cmd.Use != "app" {
		t.Fatalf("expected use app, got %s", cmd.Use)
	}
	expected := map[string]bool{"upgrade": false, "install": false}
	for _, sub := range cmd.Commands() {
		if _, ok := expected[sub.Name()]; ok {
			expected[sub.Name()] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Fatalf("expected %s subcommand to be registered", name)
		}
	}
}

func TestNewUpgradeCommandFlags(t *testing.T) {
	cmd := appcmd.NewUpgradeCommand()
	for _, name := range []string{"cluster-endpoint", "values-file", "values-passphrase", "bundle-path", "chart", "release-name", "app-version", "namespace", "state-file", "state-file-name", "airgapped", "output"} {
		if cmd.Flag(name) == nil {
			t.Fatalf("expected flag %s to exist", name)
		}
	}
}

func TestAppUpgradeCommand_TextSuccess(t *testing.T) {
	resolver := &resolvingStub{result: helm.ResolveResult{Source: pkgstate.ChartSource{Type: "oci", Reference: "oci://registry.example.com/apps/myapp:1.2.3", Digest: "sha256:abc"}}}
	stateMgr := &stateStub{path: "/var/lib/chainctl/state.json"}
	installer := &fakeHelmInstaller{}
	deps := appcmd.UpgradeDeps{
		Installer:        installer,
		TelemetryEmitter: telemetryNoop,
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
		StateFilePath:    "/var/lib/chainctl/state.json",
		Output:           "text",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := appcmd.RunUpgradeForTest(cmd, opts, deps); err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}

	if !resolver.called {
		t.Fatalf("expected resolver to be invoked")
	}
	if !stateMgr.called {
		t.Fatalf("expected state manager to persist state")
	}
	expected := "Upgrade completed successfully for release myapp-demo in namespace demo\nState written to /var/lib/chainctl/state.json\n"
	if !strings.Contains(out.String(), expected) {
		t.Fatalf("expected output to include:\n%s\nactual:\n%s", expected, out.String())
	}
}

func TestAppUpgradeCommand_JSONOutput(t *testing.T) {
	resolver := &resolvingStub{result: helm.ResolveResult{Source: pkgstate.ChartSource{Type: "oci", Reference: "oci://registry.example.com/apps/myapp:1.2.3"}}}
	stateMgr := &stateStub{path: "/var/lib/chainctl/state.json"}
	deps := appcmd.UpgradeDeps{
		Installer:        &fakeHelmInstaller{},
		TelemetryEmitter: telemetryNoop,
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
		StateFilePath:    "/var/lib/chainctl/state.json",
		Output:           "json",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)

	if err := appcmd.RunUpgradeForTest(cmd, opts, deps); err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}

	if !bytes.Contains(out.Bytes(), []byte("\"stateFile\":\"/var/lib/chainctl/state.json\"")) {
		t.Fatalf("expected json to include state file path, got %s", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("\"action\":\"upgrade\"")) {
		t.Fatalf("expected action to equal upgrade, got %s", out.String())
	}
}

func TestAppUpgradeCommand_UnsupportedOutput(t *testing.T) {
	resolver := &resolvingStub{result: helm.ResolveResult{Source: pkgstate.ChartSource{Type: "oci", Reference: "oci://registry.example.com/apps/myapp:1.2.3"}}}
	stateMgr := &stateStub{path: "/var/lib/chainctl/state.json"}
	deps := appcmd.UpgradeDeps{
		Installer:        &fakeHelmInstaller{},
		TelemetryEmitter: telemetryNoop,
		Resolver:         resolver,
		StateManager:     stateMgr,
	}

	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		ChartReference:   "oci://registry.example.com/apps/myapp:1.2.3",
		StateFilePath:    "/var/lib/chainctl/state.json",
		Output:           "yaml",
	}

	err := appcmd.RunUpgradeForTest(&cobra.Command{}, opts, deps)
	if !errors.Is(err, appcmd.ErrUnsupportedOutput()) {
		t.Fatalf("expected unsupported output error, got %v", err)
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

func TestAppUpgradeCommand_AirgappedLoadsBundle(t *testing.T) {
	var called bool
	stateMgr := &stateStub{path: "/var/lib/chainctl/state.json"}
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
		StateManager: stateMgr,
	}

	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		BundlePath:       "/mnt/package.tar",
		ReleaseName:      "myapp-demo",
		Namespace:        "demo",
		StateFilePath:    "/var/lib/chainctl/state.json",
		Output:           "text",
	}

	if err := appcmd.RunUpgradeForTest(&cobra.Command{}, opts, deps); err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}
	if !called {
		t.Fatalf("expected bundle loader to be called")
	}
	if !stateMgr.called {
		t.Fatalf("expected state manager to persist state")
	}
}

func TestAppUpgradeCommand_MutuallyExclusiveSources(t *testing.T) {
	deps := appcmd.UpgradeDeps{Installer: &fakeHelmInstaller{}, Resolver: &resolvingStub{}}
	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		ChartReference:   "oci://example.com/app:1.0.0",
		BundlePath:       "/tmp/bundle.tar",
	}

	err := appcmd.RunUpgradeForTest(&cobra.Command{}, opts, deps)
	if !errors.Is(err, appcmd.ErrConflictingSources()) {
		t.Fatalf("expected conflicting sources error, got %v", err)
	}
}

func TestAppUpgradeCommand_MissingSource(t *testing.T) {
	deps := appcmd.UpgradeDeps{Installer: &fakeHelmInstaller{}}
	opts := appcmd.UpgradeOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
	}

	err := appcmd.RunUpgradeForTest(&cobra.Command{}, opts, deps)
	if !errors.Is(err, appcmd.ErrMissingSource()) {
		t.Fatalf("expected missing source error, got %v", err)
	}
}
