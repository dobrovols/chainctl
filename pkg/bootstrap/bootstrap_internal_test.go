package bootstrap

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultRunner(t *testing.T) {
	if err := (defaultRunner{}).Run([]string{"/bin/sh", "-c", "exit 0"}, map[string]string{"TEST_ENV": "value"}); err != nil {
		t.Fatalf("expected command to succeed, got %v", err)
	}
}

func TestDefaultRunnerRequiresCommand(t *testing.T) {
	if err := (defaultRunner{}).Run(nil, nil); err == nil {
		t.Fatalf("expected error for empty command")
	}
}

func TestDefaultWaiterSleeps(t *testing.T) {
	start := time.Now()
	if err := (defaultWaiter{}).Wait(10 * time.Millisecond); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if time.Since(start) < 2*time.Second {
		t.Fatalf("expected wait to sleep, duration %v", time.Since(start))
	}
}

func TestEnvMap(t *testing.T) {
	env := envMap(map[string]string{"FOO": "bar", "BAZ": "qux"})
	if len(env) != 2 {
		t.Fatalf("expected two env entries, got %d", len(env))
	}
	joined := strings.Join(env, " ")
	if !strings.Contains(joined, "FOO=bar") {
		t.Fatalf("expected FOO entry, got %s", joined)
	}
}
