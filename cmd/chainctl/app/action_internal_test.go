package app

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/helm"
	pkgstate "github.com/dobrovols/chainctl/pkg/state"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

const (
	appTestOCIChartRef     = "oci://registry.local/my/app:1.2.3"
	appTestBundlePath      = "/bundle"
	appTestBundleTar       = "/tmp/bundle.tar"
	appTestBundleTgz       = "/tmp/bundle.tgz"
	appTestValuesFile      = "/tmp/values.enc"
	appTestSecret          = "secret"
	appTestNamespace       = "demo"
	appTestRelease         = "demo-release"
	appTestClusterEndpoint = "https://cluster.local"
	appTestLoadFailed      = "load failed"
	appTestStatePath       = "/tmp/state.json"
)

type stubResolver struct {
	called []helm.ResolveOptions
	result helm.ResolveResult
	err    error
}

func (s *stubResolver) Resolve(_ context.Context, opts helm.ResolveOptions) (helm.ResolveResult, error) {
	s.called = append(s.called, opts)
	if s.err != nil {
		return helm.ResolveResult{}, s.err
	}
	return s.result, nil
}

type capturingLoader struct {
	requests [][2]string
	bundle   *bundle.Bundle
	err      error
}

func (c *capturingLoader) load(path, cache string) (*bundle.Bundle, error) {
	c.requests = append(c.requests, [2]string{path, cache})
	if c.err != nil {
		return nil, c.err
	}
	return c.bundle, nil
}

func TestResolveChartSourceRequiresSingleSource(t *testing.T) {
	ctx := context.Background()
	deps := UpgradeDeps{}
	options := sharedOptions{ChartReference: "oci://chart", BundlePath: appTestBundleTgz}

	if _, err := resolveChartSource(ctx, options, deps); !errors.Is(err, errConflictingSources) {
		t.Fatalf("expected conflicting sources error, got %v", err)
	}

	if _, err := resolveChartSource(ctx, sharedOptions{}, deps); !errors.Is(err, errMissingSource) {
		t.Fatalf("expected missing source error, got %v", err)
	}
}

func TestResolveChartSourceUsesResolverForOCI(t *testing.T) {
	ctx := context.Background()
	options := sharedOptions{ChartReference: appTestOCIChartRef}
	res := &stubResolver{result: helm.ResolveResult{Source: pkgstate.ChartSource{Type: "oci", Reference: options.ChartReference, Digest: "sha256:digest"}}}
	deps := UpgradeDeps{Resolver: res}

	result, err := resolveChartSource(ctx, options, deps)
	if err != nil {
		t.Fatalf("resolve chart source: %v", err)
	}
	if len(res.called) != 1 {
		t.Fatalf("expected resolver to be invoked once, got %d", len(res.called))
	}
	if res.called[0].OCIReference != options.ChartReference {
		t.Fatalf("expected resolver to receive OCI reference %s, got %s", options.ChartReference, res.called[0].OCIReference)
	}
	if result.Bundle != nil {
		t.Fatalf("expected no bundle for oci source")
	}
}

func TestResolveChartSourceRequiresResolverForOCI(t *testing.T) {
	ctx := context.Background()
	options := sharedOptions{ChartReference: appTestOCIChartRef}

	if _, err := resolveChartSource(ctx, options, UpgradeDeps{}); !errors.Is(err, errResolverPullerMissing) {
		t.Fatalf("expected resolver missing error, got %v", err)
	}
}

func TestResolveChartSourcePrefersResolverBundleResult(t *testing.T) {
	ctx := context.Background()
	b := &bundle.Bundle{Path: appTestBundlePath}
	res := &stubResolver{result: helm.ResolveResult{Source: pkgstate.ChartSource{Type: "bundle", Reference: appTestBundlePath}, Bundle: b}}
	options := sharedOptions{BundlePath: appTestBundlePath}

	result, err := resolveChartSource(ctx, options, UpgradeDeps{Resolver: res})
	if err != nil {
		t.Fatalf("resolve chart source: %v", err)
	}
	if result.Bundle != b {
		t.Fatalf("expected resolver provided bundle to be used")
	}
}

func TestResolveChartSourceFallsBackToLoader(t *testing.T) {
	ctx := context.Background()
	b := &bundle.Bundle{Path: appTestBundlePath, CacheRoot: "/cache"}
	loader := &capturingLoader{bundle: b}
	options := sharedOptions{BundlePath: filepath.Join("/tmp", "bundle.tar"), ChartReference: ""}

	deps := UpgradeDeps{BundleLoader: loader.load}
	result, err := resolveChartSource(ctx, options, deps)
	if err != nil {
		t.Fatalf("resolve chart source: %v", err)
	}
	if len(loader.requests) != 1 {
		t.Fatalf("expected loader to be invoked once, got %d", len(loader.requests))
	}
	if loader.requests[0][0] != options.BundlePath {
		t.Fatalf("expected loader to receive bundle path %s, got %s", options.BundlePath, loader.requests[0][0])
	}
	if result.Bundle != b {
		t.Fatalf("expected loader provided bundle to be returned")
	}
	if result.Outcome.Source.Type != "bundle" {
		t.Fatalf("expected source type bundle, got %s", result.Outcome.Source.Type)
	}
}

func TestResolveChartSourceLoaderErrorPropagates(t *testing.T) {
	ctx := context.Background()
	loader := &capturingLoader{err: errors.New(appTestLoadFailed)}
	options := sharedOptions{BundlePath: appTestBundleTar}

	_, err := resolveChartSource(ctx, options, UpgradeDeps{BundleLoader: loader.load})
	if err == nil || !strings.Contains(err.Error(), appTestLoadFailed) {
		t.Fatalf("expected loader error, got %v", err)
	}
}

func TestResolveStateOverridesUsesResolver(t *testing.T) {
	overrides, hint, err := resolveStateOverrides(sharedOptions{StateFilePath: appTestStatePath})
	if err != nil {
		t.Fatalf("resolve state overrides: %v", err)
	}
	if overrides.StateFilePath != appTestStatePath {
		t.Fatalf("expected overrides path %s, got %s", appTestStatePath, overrides.StateFilePath)
	}
	if hint != appTestStatePath {
		t.Fatalf("expected hint %s, got %s", appTestStatePath, hint)
	}
}

func TestTelemetryEmitterDefaultEmitsEvents(t *testing.T) {
	var buf bytes.Buffer
	emitter, err := telemetryEmitterDefault(&buf)
	if err != nil {
		t.Fatalf("expected emitter without error, got %v", err)
	}
	if emitter == nil {
		t.Fatal("expected emitter instance")
	}
	if err := emitter.Emit(telemetry.Event{}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("expected telemetry emitter to write output")
	}
}

func TestEnsureDepsSetsDefaults(t *testing.T) {
	deps := UpgradeDeps{}
	ensureDeps(&deps)
	if deps.BundleLoader == nil {
		t.Fatal("expected default bundle loader")
	}
	if deps.TelemetryEmitter == nil {
		t.Fatal("expected default telemetry emitter")
	}
	if deps.StateManager == nil {
		t.Fatal("expected default state manager")
	}
	if deps.Resolver == nil {
		t.Fatal("expected default resolver")
	}
}

func TestSplitOCIReferenceParsesVersion(t *testing.T) {
	chart, version := splitOCIReference("oci://registry.local/apps/app:1.2.3")
	if version != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %s", version)
	}
	if chart != "oci://registry.local/apps/app" {
		t.Fatalf("unexpected chart reference %s", chart)
	}

	chart, version = splitOCIReference("oci://registry.local:5000/apps/app")
	if version != "" {
		t.Fatalf("expected empty version when colon belongs to host, got %s", version)
	}
	if chart != "oci://registry.local:5000/apps/app" {
		t.Fatalf("unexpected chart reference %s", chart)
	}
}

func TestBuildProfileForActionInstall(t *testing.T) {
	opts := sharedOptions{
		ValuesFile:       appTestValuesFile,
		ValuesPassphrase: appTestSecret,
		Namespace:        appTestNamespace,
		ReleaseName:      appTestRelease,
	}
	profile, err := buildProfileForAction(opts, actionInstall)
	if err != nil {
		t.Fatalf("buildProfileForAction: %v", err)
	}
	if profile.HelmNamespace != appTestNamespace {
		t.Fatalf("expected namespace %s, got %s", appTestNamespace, profile.HelmNamespace)
	}
	if profile.HelmRelease != appTestRelease {
		t.Fatalf("expected release %s, got %s", appTestRelease, profile.HelmRelease)
	}
}

func TestBuildProfileForActionUpgradeRequiresEndpoint(t *testing.T) {
	_, err := buildProfileForAction(sharedOptions{ValuesFile: appTestValuesFile}, actionUpgrade)
	if err == nil {
		t.Fatalf("expected error when upgrade lacks cluster endpoint")
	}

	profile, err := buildProfileForAction(sharedOptions{
		ClusterEndpoint: appTestClusterEndpoint,
		ValuesFile:      appTestValuesFile,
	}, actionUpgrade)
	if err != nil {
		t.Fatalf("buildProfileForAction upgrade: %v", err)
	}
	if profile.ClusterEndpoint != appTestClusterEndpoint {
		t.Fatalf("expected cluster endpoint to be set, got %s", profile.ClusterEndpoint)
	}
}

func TestEmitOutputUnsupportedFormat(t *testing.T) {
	cmd := &cobra.Command{}
	err := emitOutput(cmd, &config.Profile{}, helm.ResolveResult{}, "", "yaml", actionInstall, sharedOptions{})
	if !errors.Is(err, errUnsupportedOutput) {
		t.Fatalf("expected errUnsupportedOutput, got %v", err)
	}
}

func TestNewOCIPullerConstructsInstance(t *testing.T) {
	puller, err := newOCIPuller()
	if err != nil {
		t.Fatalf("expected puller without error, got %v", err)
	}
	if puller == nil {
		t.Fatal("expected non-nil puller")
	}
}

func TestRunAppActionRequiresValuesFile(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	deps := UpgradeDeps{StateManager: pkgstate.NewManager(nil)}

	if err := runAppAction(cmd, sharedOptions{}, deps, actionInstall); !errors.Is(err, errValuesFile) {
		t.Fatalf("expected values file error, got %v", err)
	}
}
