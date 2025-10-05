package unit

import (
	"bytes"
	"errors"
	"io"
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

func TestClusterInstallCommand_TextSuccess(t *testing.T) {
	inspector := stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true}, sudo: true}
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
	inspector := stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true}, sudo: true}
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
	inspector := stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true}, sudo: true}

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
	inspector := stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true}, sudo: true}
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
		Inspector: stubInspector{cpu: 8, memory: 16, modules: map[string]bool{"br_netfilter": true}, sudo: true},
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
