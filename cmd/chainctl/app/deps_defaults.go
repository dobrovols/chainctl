package app

import (
	"context"
	"strings"

	"github.com/dobrovols/chainctl/internal/state"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/helm"
	pkgstate "github.com/dobrovols/chainctl/pkg/state"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

var defaultUpgradeDeps UpgradeDeps

type ociPuller struct {
	settings *cli.EnvSettings
}

func newOCIPuller() (helm.OCIPuller, error) {
	return &ociPuller{settings: cli.New()}, nil
}

func (p *ociPuller) Pull(ctx context.Context, ref string) (helm.PullResult, error) {
	chartRef, version := splitOCIReference(ref)
	opts := action.ChartPathOptions{PassCredentialsAll: true}
	if version != "" {
		opts.Version = version
	}
	path, err := opts.LocateChart(chartRef, p.settings)
	if err != nil {
		return helm.PullResult{}, err
	}

	// Attempt to extract digest from the ref (e.g., ...@sha256:...)
	var digest string
	if atIdx := strings.LastIndex(ref, "@sha256:"); atIdx != -1 {
		digest = ref[atIdx+1:]
	}

	return helm.PullResult{
		ChartPath: path,
		Digest:    digest,
	}, nil
}

func splitOCIReference(ref string) (string, string) {
	idx := strings.LastIndex(ref, ":")
	if idx == -1 {
		return ref, ""
	}
	lastSlash := strings.LastIndex(ref, "/")
	if idx > lastSlash {
		return ref[:idx], ref[idx+1:]
	}
	return ref, ""
}

func ensureDeps(deps *UpgradeDeps) {
	if deps.BundleLoader == nil {
		deps.BundleLoader = bundle.Load
	}
	if deps.TelemetryEmitter == nil {
		deps.TelemetryEmitter = telemetryEmitterDefault
	}
	if deps.StateManager == nil {
		deps.StateManager = pkgstate.NewManager(state.NewResolver())
	}
	if deps.Resolver == nil {
		puller, err := newOCIPuller()
		if err != nil {
			deps.Resolver = helm.NewResolver(nil, deps.BundleLoader)
		} else {
			deps.Resolver = helm.NewResolver(puller, deps.BundleLoader)
		}
	}
}
