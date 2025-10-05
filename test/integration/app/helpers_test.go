package appintegration_test

import (
	"io"

	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

type noopInstaller struct{}

func (noopInstaller) Install(*config.Profile, *bundle.Bundle) error { return nil }

func telemetrySilentEmitter(io.Writer) *telemetry.Emitter {
	return telemetry.NewEmitter(io.Discard)
}
