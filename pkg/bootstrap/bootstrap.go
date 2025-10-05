package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/dobrovols/chainctl/internal/config"
)

// Runner executes bootstrap commands.
type Runner interface {
	Run(cmd []string, env map[string]string) error
}

// Waiter waits for cluster readiness after bootstrap.
type Waiter interface {
	Wait(timeout time.Duration) error
}

// Orchestrator controls the bootstrap workflow.
type Orchestrator struct {
	runner  Runner
	waiter  Waiter
	timeout time.Duration
}

// NewOrchestrator constructs an orchestrator with the given runner and waiter.
func NewOrchestrator(r Runner, w Waiter) *Orchestrator {
	if r == nil {
		r = defaultRunner{}
	}
	if w == nil {
		w = defaultWaiter{}
	}
	return &Orchestrator{runner: r, waiter: w, timeout: 10 * time.Minute}
}

// Bootstrap executes the k3s bootstrap flow if the profile requests it.
func (o *Orchestrator) Bootstrap(profile *config.Profile) error {
	if profile.Mode != config.ModeBootstrap {
		return nil
	}

	env := map[string]string{
		"INSTALL_K3S_CHANNEL": profile.K3sVersion,
		"INSTALL_K3S_EXEC":    "server --write-kubeconfig-mode=644 --disable traefik",
	}

	scriptSHA := os.Getenv("CHAINCTL_K3S_INSTALL_SHA256")
	if scriptSHA == "" {
		return errors.New("CHAINCTL_K3S_INSTALL_SHA256 must be set for secure k3s bootstrap")
	}

    if scriptPath := os.Getenv("CHAINCTL_K3S_INSTALL_PATH"); scriptPath != "" {
        if _, err := os.Stat(scriptPath); err != nil {
            return fmt.Errorf("invalid CHAINCTL_K3S_INSTALL_PATH: %w", err)
        }
        command := fmt.Sprintf("set -euo pipefail; printf '%%s  %%s\\n' '%s' '%s' | sha256sum -c -; sh '%s'", scriptSHA, scriptPath, scriptPath)
        cmd := []string{"sh", "-c", command}
        if err := o.runner.Run(cmd, env); err != nil {
            return err
        }
        return o.waiter.Wait(o.timeout)
    }

	scriptURL := os.Getenv("CHAINCTL_K3S_INSTALL_URL")
	if scriptURL == "" {
		return errors.New("set CHAINCTL_K3S_INSTALL_URL or CHAINCTL_K3S_INSTALL_PATH")
	}

    command := fmt.Sprintf("set -euo pipefail; tmp=$(mktemp); trap 'rm -f $tmp' EXIT; curl -sfL '%s' -o \"$tmp\"; printf '%%s  %%s\\n' '%s' \"$tmp\" | sha256sum -c -; sh \"$tmp\"", scriptURL, scriptSHA)
    cmd := []string{"sh", "-c", command}

	if err := o.runner.Run(cmd, env); err != nil {
		return err
	}

	return o.waiter.Wait(o.timeout)
}

type defaultRunner struct{}

type execRunner struct{}

func (defaultRunner) Run(cmd []string, env map[string]string) error {
	if len(cmd) == 0 {
		return fmt.Errorf("no command provided")
	}
	command := exec.CommandContext(context.Background(), cmd[0], cmd[1:]...)
	command.Env = append(command.Env, envMap(env)...)
	command.Stdout = nil
	command.Stderr = nil
	return command.Run()
}

type defaultWaiter struct{}

func (defaultWaiter) Wait(timeout time.Duration) error {
	time.Sleep(2 * time.Second)
	return nil
}

func envMap(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}
	return out
}
