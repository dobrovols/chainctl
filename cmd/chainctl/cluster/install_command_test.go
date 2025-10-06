package cluster_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"

	clustercmd "github.com/dobrovols/chainctl/cmd/chainctl/cluster"
	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

type stubInspector struct {
	cpu     int
	memory  int
	modules map[string]bool
	sudo    bool
}

func (s stubInspector) CPUCount() int                 { return s.cpu }
func (s stubInspector) MemoryGiB() int                { return s.memory }
func (s stubInspector) HasKernelModule(m string) bool { return s.modules[m] }
func (s stubInspector) HasSudoPrivileges() bool       { return s.sudo }

type fakeBootstrap struct {
	called  bool
	profile *config.Profile
	err     error
}

func (f *fakeBootstrap) Bootstrap(p *config.Profile) error {
	f.called = true
	f.profile = p
	return f.err
}

type fakeHelm struct {
	called  bool
	profile *config.Profile
	bundle  *bundle.Bundle
	err     error
}

func (f *fakeHelm) Install(p *config.Profile, b *bundle.Bundle) error {
	f.called = true
	f.profile = p
	f.bundle = b
	return f.err
}

func telemetryStub(w io.Writer) *telemetry.Emitter {
	return telemetry.NewEmitter(w)
}

func TestNewInstallCommandRegistersFlags(t *testing.T) {
	cmd := clustercmd.NewInstallCommand()
	for _, name := range []string{
		"bootstrap", "cluster-endpoint", "k3s-version", "values-file", "values-passphrase", "bundle-path", "airgapped", "dry-run", "output",
	} {
		if cmd.Flag(name) == nil {
			t.Fatalf("expected flag %s to be defined", name)
		}
	}
}

func TestInstallCommandSentinelsExposeErrors(t *testing.T) {
	if clustercmd.ErrBundleRequired() == nil {
		t.Fatalf("expected ErrBundleRequired to return non-nil error")
	}
	if clustercmd.ErrValuesFileRequired() == nil {
		t.Fatalf("expected ErrValuesFileRequired to return non-nil error")
	}
	if clustercmd.ErrUnsupportedOutput() == nil {
		t.Fatalf("expected ErrUnsupportedOutput to return non-nil error")
	}
}

func TestClusterInstallCommand_TextSuccess(t *testing.T) {
	inspector := stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true, "overlay": true}, sudo: true}
	bootstrap := &fakeBootstrap{}
	helm := &fakeHelm{}

	deps := clustercmd.InstallDeps{
		Inspector: inspector,
		BundleLoader: func(path, cache string) (*bundle.Bundle, error) {
			return &bundle.Bundle{Manifest: bundle.Manifest{Version: "1.0.0"}}, nil
		},
		Bootstrapper:        bootstrap,
		HelmInstaller:       helm,
		TelemetryEmitter:    telemetryStub,
		ClusterValidator:    func(*rest.Config) error { return nil },
		ClusterConfigLoader: func(*config.Profile) (*rest.Config, error) { return nil, nil },
	}

	opts := clustercmd.InstallOptions{
		Bootstrap:        true,
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		Output:           "text",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := clustercmd.RunInstallForTest(cmd, opts, deps); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if !bootstrap.called {
		t.Fatalf("expected bootstrapper to be called")
	}
	if !helm.called {
		t.Fatalf("expected helm installer to be called")
	}
	if !bytes.Contains(out.Bytes(), []byte("Installation completed")) {
		t.Fatalf("expected success message, got %s", out.String())
	}
}

func TestClusterInstallCommand_DryRunSkipsBootstrap(t *testing.T) {
	inspector := stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true, "overlay": true}, sudo: true}
	bootstrap := &fakeBootstrap{}
	helm := &fakeHelm{}

	deps := clustercmd.InstallDeps{
		Inspector: inspector,
		BundleLoader: func(path, cache string) (*bundle.Bundle, error) {
			return &bundle.Bundle{Manifest: bundle.Manifest{Version: "1.0.0"}}, nil
		},
		Bootstrapper:        bootstrap,
		HelmInstaller:       helm,
		TelemetryEmitter:    telemetryStub,
		ClusterValidator:    func(*rest.Config) error { return nil },
		ClusterConfigLoader: func(*config.Profile) (*rest.Config, error) { return nil, nil },
	}

	opts := clustercmd.InstallOptions{
		Bootstrap:        true,
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		DryRun:           true,
		Output:           "text",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := clustercmd.RunInstallForTest(cmd, opts, deps); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if bootstrap.called {
		t.Fatalf("expected bootstrapper to be skipped in dry-run")
	}
	if helm.called {
		t.Fatalf("expected helm installer to be skipped in dry-run")
	}
	if !bytes.Contains(out.Bytes(), []byte("Dry-run")) {
		t.Fatalf("expected dry-run message, got %s", out.String())
	}
}

func TestClusterInstallCommand_JSONOutput(t *testing.T) {
	inspector := stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true, "overlay": true}, sudo: true}

	deps := clustercmd.InstallDeps{
		Inspector: inspector,
		BundleLoader: func(path, cache string) (*bundle.Bundle, error) {
			return &bundle.Bundle{Manifest: bundle.Manifest{Version: "1.0.0"}}, nil
		},
		Bootstrapper:        &fakeBootstrap{},
		HelmInstaller:       &fakeHelm{},
		TelemetryEmitter:    telemetryStub,
		ClusterValidator:    func(*rest.Config) error { return nil },
		ClusterConfigLoader: func(*config.Profile) (*rest.Config, error) { return nil, nil },
	}

	opts := clustercmd.InstallOptions{
		Bootstrap:        true,
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		Output:           "json",
	}

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := clustercmd.RunInstallForTest(cmd, opts, deps); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if !bytes.Contains(out.Bytes(), []byte("\"status\":\"success\"")) {
		t.Fatalf("expected json success output, got %s", out.String())
	}
}

func TestClusterInstallCommand_AirgappedRequiresBundle(t *testing.T) {
	inspector := stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true, "overlay": true}, sudo: true}
	deps := clustercmd.InstallDeps{
		Inspector: inspector,
		BundleLoader: func(path, cache string) (*bundle.Bundle, error) {
			return &bundle.Bundle{Manifest: bundle.Manifest{Version: "1.0.0"}}, nil
		},
		Bootstrapper:        &fakeBootstrap{},
		HelmInstaller:       &fakeHelm{},
		TelemetryEmitter:    telemetryStub,
		ClusterValidator:    func(*rest.Config) error { return nil },
		ClusterConfigLoader: func(*config.Profile) (*rest.Config, error) { return nil, nil },
	}

	opts := clustercmd.InstallOptions{
		Bootstrap:        true,
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		Airgapped:        true,
	}

	err := clustercmd.RunInstallForTest(&cobra.Command{}, opts, deps)
	if !errors.Is(err, config.ErrBundlePathRequired()) {
		t.Fatalf("expected bundle required error, got %v", err)
	}
}

func TestClusterInstallCommand_PreflightFailure(t *testing.T) {
	inspector := stubInspector{cpu: 1, memory: 1, modules: map[string]bool{}, sudo: false}
	deps := clustercmd.InstallDeps{
		Inspector: inspector,
		BundleLoader: func(path, cache string) (*bundle.Bundle, error) {
			return &bundle.Bundle{Manifest: bundle.Manifest{Version: "1.0.0"}}, nil
		},
		Bootstrapper:        &fakeBootstrap{},
		HelmInstaller:       &fakeHelm{},
		TelemetryEmitter:    telemetryStub,
		ClusterValidator:    func(*rest.Config) error { return nil },
		ClusterConfigLoader: func(*config.Profile) (*rest.Config, error) { return nil, nil },
	}

	opts := clustercmd.InstallOptions{
		Bootstrap:        true,
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
	}

	err := clustercmd.RunInstallForTest(&cobra.Command{}, opts, deps)
	if err == nil {
		t.Fatalf("expected preflight failure")
	}
}

func TestClusterInstallCommand_ReuseValidatesCluster(t *testing.T) {
	validatorCalled := false
	loaderCalled := false
	deps := clustercmd.InstallDeps{
		Inspector: stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true, "overlay": true}, sudo: true},
		BundleLoader: func(path, cache string) (*bundle.Bundle, error) {
			return nil, nil
		},
		Bootstrapper:     &fakeBootstrap{},
		HelmInstaller:    &fakeHelm{},
		TelemetryEmitter: telemetryStub,
		ClusterConfigLoader: func(*config.Profile) (*rest.Config, error) {
			loaderCalled = true
			return &rest.Config{}, nil
		},
		ClusterValidator: func(cfg *rest.Config) error {
			if cfg == nil {
				t.Fatalf("expected non-nil rest config")
			}
			validatorCalled = true
			return nil
		},
	}

	opts := clustercmd.InstallOptions{
		Bootstrap:        false,
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		Output:           "text",
	}

	cmd := &cobra.Command{}
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	if err := clustercmd.RunInstallForTest(cmd, opts, deps); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	if !loaderCalled {
		t.Fatalf("expected cluster config loader to be called")
	}
	if !validatorCalled {
		t.Fatalf("expected cluster validator to be called")
	}
}

func TestClusterInstallCommand_UnsupportedOutput(t *testing.T) {
	deps := clustercmd.InstallDeps{
		Inspector:        stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true, "overlay": true}, sudo: true},
		Bootstrapper:     &fakeBootstrap{},
		HelmInstaller:    &fakeHelm{},
		TelemetryEmitter: telemetryStub,
	}

	opts := clustercmd.InstallOptions{
		Bootstrap:        true,
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		Output:           "yaml",
	}

	err := clustercmd.RunInstallForTest(&cobra.Command{}, opts, deps)
	if !errors.Is(err, clustercmd.ErrUnsupportedOutput()) {
		t.Fatalf("expected unsupported output error, got %v", err)
	}
}

func TestClusterInstallCommand_AirgappedLoadsBundle(t *testing.T) {
	var called bool
	deps := clustercmd.InstallDeps{
		Inspector: stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true, "overlay": true}, sudo: true},
		BundleLoader: func(path, cache string) (*bundle.Bundle, error) {
			called = true
			if path != "/mnt/airgap.tar" {
				t.Fatalf("unexpected bundle path %s", path)
			}
			if cache == "" {
				t.Fatalf("expected cache directory")
			}
			return &bundle.Bundle{Manifest: bundle.Manifest{Version: "1.0.0"}}, nil
		},
		Bootstrapper:     &fakeBootstrap{},
		HelmInstaller:    &fakeHelm{},
		TelemetryEmitter: telemetryStub,
	}

	opts := clustercmd.InstallOptions{
		Bootstrap:        true,
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		Airgapped:        true,
		BundlePath:       "/mnt/airgap.tar",
		Output:           "text",
	}

	cmd := &cobra.Command{}
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	if err := clustercmd.RunInstallForTest(cmd, opts, deps); err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if !called {
		t.Fatalf("expected bundle loader to be invoked")
	}
}

func TestClusterInstallCommand_LoadClusterConfigError(t *testing.T) {
	deps := clustercmd.InstallDeps{
		Inspector:        stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true, "overlay": true}, sudo: true},
		BundleLoader:     func(string, string) (*bundle.Bundle, error) { return nil, nil },
		Bootstrapper:     &fakeBootstrap{},
		HelmInstaller:    &fakeHelm{},
		TelemetryEmitter: telemetryStub,
	}

	// Ensure default loader runs and fails by pointing kubeconfig to a missing file.
	original := os.Getenv("KUBECONFIG")
	missing := filepath.Join(t.TempDir(), "missing-kubeconfig")
	os.Setenv("KUBECONFIG", missing)
	defer os.Setenv("KUBECONFIG", original)

	opts := clustercmd.InstallOptions{
		Bootstrap:        false,
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		Output:           "text",
	}

	err := clustercmd.RunInstallForTest(&cobra.Command{}, opts, deps)
	if err == nil || !strings.Contains(err.Error(), "load cluster config") {
		t.Fatalf("expected cluster config error, got %v", err)
	}
}
