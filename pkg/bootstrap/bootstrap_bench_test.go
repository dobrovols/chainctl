package bootstrap

import (
	"testing"
	"time"

	"github.com/dobrovols/chainctl/internal/config"
)

type benchRunner struct{}

type benchWaiter struct{}

func (benchRunner) Run(cmd []string, env map[string]string) error { return nil }
func (benchWaiter) Wait(timeout time.Duration) error              { return nil }

func BenchmarkBootstrap(b *testing.B) {
	orch := NewOrchestrator(benchRunner{}, benchWaiter{})
	profile := &config.Profile{Mode: config.ModeBootstrap, K3sVersion: "v1.30.2"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := orch.Bootstrap(profile); err != nil {
			b.Fatalf("bootstrap: %v", err)
		}
	}
}
