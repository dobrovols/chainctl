package bootstrap_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bootstrap"
)

type fakeRunner struct {
	cmd []string
	env map[string]string
	err error
}

func (f *fakeRunner) Run(cmd []string, env map[string]string) error {
	f.cmd = cmd
	f.env = env
	return f.err
}

type fakeWaiter struct {
	waited bool
	err    error
}

func (f *fakeWaiter) Wait(timeout time.Duration) error {
	f.waited = true
	return f.err
}

func TestBootstrapExecutesInstaller(t *testing.T) {
	t.Setenv("CHAINCTL_K3S_INSTALL_URL", "https://example.com/install.sh")
	t.Setenv("CHAINCTL_K3S_INSTALL_SHA256", "deadbeefcafebabe")

	runner := &fakeRunner{}
	waiter := &fakeWaiter{}
	orch := bootstrap.NewOrchestrator(runner, waiter)

	profile := &config.Profile{Mode: config.ModeBootstrap, K3sVersion: "v1.30.2"}
	if err := orch.Bootstrap(profile); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	if len(runner.cmd) == 0 {
		t.Fatalf("expected runner to be invoked")
	}
	if !strings.Contains(runner.cmd[2], "sha256sum -c -") {
		t.Fatalf("expected checksum verification in command, got %s", runner.cmd[2])
	}
	if _, ok := runner.env["INSTALL_K3S_CHANNEL"]; !ok {
		t.Fatalf("expected INSTALL_K3S_CHANNEL env to be set")
	}
	if !waiter.waited {
		t.Fatalf("expected waiter to be invoked")
	}
}

func TestBootstrapSkipsWhenReuseMode(t *testing.T) {
	runner := &fakeRunner{}
	waiter := &fakeWaiter{}
	orch := bootstrap.NewOrchestrator(runner, waiter)

	profile := &config.Profile{Mode: config.ModeReuse}
	if err := orch.Bootstrap(profile); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	if runner.cmd != nil {
		t.Fatalf("expected runner not to be called")
	}
	if waiter.waited {
		t.Fatalf("expected waiter not to be called")
	}
}

func TestBootstrapPropagatesRunnerError(t *testing.T) {
	t.Setenv("CHAINCTL_K3S_INSTALL_URL", "https://example.com/install.sh")
	t.Setenv("CHAINCTL_K3S_INSTALL_SHA256", "deadbeefcafebabe")

	wantErr := errors.New("exec failed")
	runner := &fakeRunner{err: wantErr}
	waiter := &fakeWaiter{}
	orch := bootstrap.NewOrchestrator(runner, waiter)

	profile := &config.Profile{Mode: config.ModeBootstrap}
	err := orch.Bootstrap(profile)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected runner error, got %v", err)
	}
}

func TestBootstrapPropagatesWaitError(t *testing.T) {
	t.Setenv("CHAINCTL_K3S_INSTALL_URL", "https://example.com/install.sh")
	t.Setenv("CHAINCTL_K3S_INSTALL_SHA256", "deadbeefcafebabe")

	wantErr := errors.New("not ready")
	runner := &fakeRunner{}
	waiter := &fakeWaiter{err: wantErr}
	orch := bootstrap.NewOrchestrator(runner, waiter)

	profile := &config.Profile{Mode: config.ModeBootstrap}
	err := orch.Bootstrap(profile)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected waiter error, got %v", err)
	}
}

func TestBootstrapRequiresSHA(t *testing.T) {
	orch := bootstrap.NewOrchestrator(&fakeRunner{}, &fakeWaiter{})
	err := orch.Bootstrap(&config.Profile{Mode: config.ModeBootstrap})
	if err == nil {
		t.Fatalf("expected error when SHA not provided")
	}
}

func TestBootstrapLocalScriptPathValidation(t *testing.T) {
	t.Setenv("CHAINCTL_K3S_INSTALL_PATH", "/tmp/missing-install.sh")
	t.Setenv("CHAINCTL_K3S_INSTALL_SHA256", "deadbeefcafebabe")

	orch := bootstrap.NewOrchestrator(&fakeRunner{}, &fakeWaiter{})
	err := orch.Bootstrap(&config.Profile{Mode: config.ModeBootstrap})
	if err == nil {
		t.Fatalf("expected error for invalid local path")
	}
}
