package helm

import (
	"testing"

	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
)

type benchExecutor struct{}

func (benchExecutor) UpgradeRelease(*config.Profile, *bundle.Bundle) error { return nil }

func BenchmarkHelmInstall(b *testing.B) {
	installer := NewInstaller(benchExecutor{})
	profile := &config.Profile{HelmRelease: "chainapp"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := installer.Install(profile, &bundle.Bundle{}); err != nil {
			b.Fatalf("install: %v", err)
		}
	}
}
