package helm_test

import (
	"context"
	"errors"
	"testing"

	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/helm"
)

type stubPuller struct {
	ref    string
	result helm.PullResult
	err    error
	called bool
}

func (s *stubPuller) Pull(ctx context.Context, ref string) (helm.PullResult, error) {
	s.called = true
	s.ref = ref
	if s.err != nil {
		return helm.PullResult{}, s.err
	}
	return s.result, nil
}

type stubBundleLoader struct {
	path      string
	cacheRoot string
	bundle    *bundle.Bundle
	err       error
	called    bool
}

func (s *stubBundleLoader) Load(path, cacheRoot string) (*bundle.Bundle, error) {
	s.called = true
	s.path = path
	s.cacheRoot = cacheRoot
	if s.err != nil {
		return nil, s.err
	}
	return s.bundle, nil
}

func TestResolverResolvesOCIReference(t *testing.T) {
	puller := &stubPuller{result: helm.PullResult{ChartPath: "/tmp/chart", Digest: "sha256:abc"}}
	loader := &stubBundleLoader{}
	resolver := helm.NewResolver(puller, loader.Load)

	ctx := context.Background()
	opts := helm.ResolveOptions{OCIReference: "oci://registry.example.com/team/app:1.2.3"}

	result, err := resolver.Resolve(ctx, opts)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if !puller.called {
		t.Fatal("expected puller to be called")
	}
	if puller.ref != opts.OCIReference {
		t.Fatalf("expected puller to receive %s, got %s", opts.OCIReference, puller.ref)
	}

	if result.Source.Type != "oci" {
		t.Fatalf("expected source type oci, got %s", result.Source.Type)
	}
	if result.Source.Reference != opts.OCIReference {
		t.Fatalf("expected source reference %s, got %s", opts.OCIReference, result.Source.Reference)
	}
	if result.Source.Digest != "sha256:abc" {
		t.Fatalf("expected digest sha256:abc, got %s", result.Source.Digest)
	}
	if result.ChartPath != puller.result.ChartPath {
		t.Fatalf("expected chart path %s, got %s", puller.result.ChartPath, result.ChartPath)
	}
	if result.Bundle != nil {
		t.Fatal("expected bundle to be nil for OCI source")
	}
}

func TestResolverResolvesBundlePath(t *testing.T) {
	puller := &stubPuller{}
	bundlePtr := &bundle.Bundle{Path: "/tmp/bundle"}
	loader := &stubBundleLoader{bundle: bundlePtr}
	resolver := helm.NewResolver(puller, loader.Load)

	ctx := context.Background()
	opts := helm.ResolveOptions{BundlePath: "/data/bundle.tar.gz", BundleCacheDir: "/cache"}

	result, err := resolver.Resolve(ctx, opts)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if loader.path != opts.BundlePath {
		t.Fatalf("expected loader to receive path %s, got %s", opts.BundlePath, loader.path)
	}
	if loader.cacheRoot != opts.BundleCacheDir {
		t.Fatalf("expected cache root %s, got %s", opts.BundleCacheDir, loader.cacheRoot)
	}

	if result.Source.Type != "bundle" {
		t.Fatalf("expected source type bundle, got %s", result.Source.Type)
	}
	if result.Source.Reference != opts.BundlePath {
		t.Fatalf("expected source reference %s, got %s", opts.BundlePath, result.Source.Reference)
	}
	if result.Bundle != bundlePtr {
		t.Fatal("expected bundle pointer to be returned")
	}
	if result.ChartPath != "" {
		t.Fatalf("expected chart path to be empty for bundle, got %s", result.ChartPath)
	}
	if puller.called {
		t.Fatal("did not expect puller to be called for bundle")
	}
}

func TestResolverErrorsWhenBothSourcesProvided(t *testing.T) {
	resolver := helm.NewResolver(&stubPuller{}, (&stubBundleLoader{}).Load)
	ctx := context.Background()
	opts := helm.ResolveOptions{OCIReference: "oci://example.com/app:1.0.0", BundlePath: "/tmp/bundle"}

	_, err := resolver.Resolve(ctx, opts)
	if err == nil {
		t.Fatal("expected error when both OCI reference and bundle path provided")
	}
}

func TestResolverErrorsWhenNoSourceProvided(t *testing.T) {
	resolver := helm.NewResolver(&stubPuller{}, (&stubBundleLoader{}).Load)
	ctx := context.Background()

	_, err := resolver.Resolve(ctx, helm.ResolveOptions{})
	if err == nil {
		t.Fatal("expected error when no chart source provided")
	}
}

func TestResolverErrorsOnInvalidOCIReference(t *testing.T) {
	resolver := helm.NewResolver(&stubPuller{}, (&stubBundleLoader{}).Load)
	ctx := context.Background()

	_, err := resolver.Resolve(ctx, helm.ResolveOptions{OCIReference: "http://not-oci"})
	if err == nil {
		t.Fatal("expected error for invalid OCI reference")
	}
}

func TestResolverSurfacesPullerErrors(t *testing.T) {
	puller := &stubPuller{err: errors.New("pull failed")}
	resolver := helm.NewResolver(puller, (&stubBundleLoader{}).Load)
	ctx := context.Background()

	_, err := resolver.Resolve(ctx, helm.ResolveOptions{OCIReference: "oci://example.com/app:1.0.0"})
	if !errors.Is(err, puller.err) {
		t.Fatalf("expected puller error to propagate, got %v", err)
	}
}

func TestResolverSurfacesBundleLoaderErrors(t *testing.T) {
	loader := &stubBundleLoader{err: errors.New("load failed")}
	resolver := helm.NewResolver(&stubPuller{}, loader.Load)
	ctx := context.Background()

	_, err := resolver.Resolve(ctx, helm.ResolveOptions{BundlePath: "/tmp/bundle"})
	if !errors.Is(err, loader.err) {
		t.Fatalf("expected loader error to propagate, got %v", err)
	}
}
