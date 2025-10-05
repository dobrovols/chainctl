package helm

import (
	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
)

// Executor abstracts helm upgrade execution.
type Executor interface {
	UpgradeRelease(*config.Profile, *bundle.Bundle) error
}

// Installer orchestrates Helm install/upgrade logic.
type Installer struct {
	exec Executor
}

// NewInstaller constructs an installer with the provided executor.
func NewInstaller(exec Executor) *Installer {
	if exec == nil {
		exec = noopExecutor{}
	}
	return &Installer{exec: exec}
}

// Install applies the Helm release according to the profile.
func (i *Installer) Install(profile *config.Profile, b *bundle.Bundle) error {
	return i.exec.UpgradeRelease(profile, b)
}

type noopExecutor struct{}

func (noopExecutor) UpgradeRelease(*config.Profile, *bundle.Bundle) error { return nil }
