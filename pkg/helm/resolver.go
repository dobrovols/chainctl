package helm

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/state"
)

// PullResult captures the outcome of fetching a chart from an OCI registry.
type PullResult struct {
	ChartPath string
	Digest    string
}

// OCIPuller downloads Helm charts from OCI registries.
type OCIPuller interface {
	Pull(ctx context.Context, ref string) (PullResult, error)
}

// BundleLoader loads a local bundle from disk.
type BundleLoader func(bundlePath, cacheRoot string) (*bundle.Bundle, error)

// ResolveOptions define the user inputs guiding chart resolution.
type ResolveOptions struct {
	OCIReference   string
	BundlePath     string
	BundleCacheDir string
}

// ResolveResult describes the selected chart source and auxiliary data required to apply it.
type ResolveResult struct {
	Source    state.ChartSource
	ChartPath string
	Bundle    *bundle.Bundle
}

// Resolver normalises user input into a chart source usable by the Helm installer.
type Resolver struct {
	puller       OCIPuller
	bundleLoader BundleLoader
}

var (
	errResolverConflictingSources  = errors.New("exactly one of --chart or --bundle-path must be provided")
	errResolverMissingSource       = errors.New("a chart reference or bundle path must be provided")
	errResolverInvalidOCI          = errors.New("invalid OCI artifact reference")
	errResolverPullerMissing       = errors.New("oci puller not configured")
	errResolverBundleLoaderMissing = errors.New("bundle loader not configured")
)

// NewResolver constructs a Resolver with the provided dependencies.
func NewResolver(puller OCIPuller, loader BundleLoader) *Resolver {
	return &Resolver{puller: puller, bundleLoader: loader}
}

// ErrResolverConflictingSources exposes the mutual exclusion error.
func ErrResolverConflictingSources() error { return errResolverConflictingSources }

// ErrResolverMissingSource exposes the missing source error.
func ErrResolverMissingSource() error { return errResolverMissingSource }

// ErrResolverInvalidOCI exposes the invalid OCI error.
func ErrResolverInvalidOCI() error { return errResolverInvalidOCI }

func (r *Resolver) Resolve(ctx context.Context, opts ResolveOptions) (ResolveResult, error) {
	hasChart := strings.TrimSpace(opts.OCIReference) != ""
	hasBundle := strings.TrimSpace(opts.BundlePath) != ""

	switch {
	case hasChart && hasBundle:
		return ResolveResult{}, errResolverConflictingSources
	case !hasChart && !hasBundle:
		return ResolveResult{}, errResolverMissingSource
	case hasChart:
		return r.resolveOCI(ctx, opts)
	default:
		return r.resolveBundle(ctx, opts)
	}
}

func (r *Resolver) resolveOCI(ctx context.Context, opts ResolveOptions) (ResolveResult, error) {
	if !strings.HasPrefix(strings.ToLower(opts.OCIReference), "oci://") {
		return ResolveResult{}, errResolverInvalidOCI
	}
	if r.puller == nil {
		return ResolveResult{}, errResolverPullerMissing
	}

	result, err := r.puller.Pull(ctx, opts.OCIReference)
	if err != nil {
		return ResolveResult{}, err
	}

	return ResolveResult{
		Source: state.ChartSource{
			Type:      "oci",
			Reference: opts.OCIReference,
			Digest:    result.Digest,
		},
		ChartPath: result.ChartPath,
	}, nil
}

func (r *Resolver) resolveBundle(_ context.Context, opts ResolveOptions) (ResolveResult, error) {
	if r.bundleLoader == nil {
		return ResolveResult{}, errResolverBundleLoaderMissing
	}
	cacheRoot := opts.BundleCacheDir
	if strings.TrimSpace(cacheRoot) == "" {
		cacheRoot = filepath.Dir(opts.BundlePath)
	}

	tb, err := r.bundleLoader(opts.BundlePath, cacheRoot)
	if err != nil {
		return ResolveResult{}, err
	}

	return ResolveResult{
		Source: state.ChartSource{
			Type:      "bundle",
			Reference: opts.BundlePath,
		},
		Bundle: tb,
	}, nil
}
