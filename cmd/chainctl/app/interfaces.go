package app

import (
	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
)

// HelmInstaller orchestrates Helm release operations.
type HelmInstaller interface {
	Install(*config.Profile, *bundle.Bundle) error
}

type noopInstaller struct{}

func (noopInstaller) Install(*config.Profile, *bundle.Bundle) error { return nil }
