package helm

import (
	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/telemetry"
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

// WithLogger returns a new installer that emits structured logs via the provided logger.
func (i *Installer) WithLogger(logger telemetry.StructuredLogger) *Installer {
	if i == nil || logger == nil {
		return i
	}
	return &Installer{exec: NewLoggingExecutor(i.exec, logger)}
}

// Install applies the Helm release according to the profile.
func (i *Installer) Install(profile *config.Profile, b *bundle.Bundle) error {
	return i.exec.UpgradeRelease(profile, b)
}

type noopExecutor struct{}

func (noopExecutor) UpgradeRelease(*config.Profile, *bundle.Bundle) error { return nil }
