package cluster

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
)

const (
	clusterTestValuesFile      = "/tmp/values.enc"
	clusterTestSecret          = "secret"
	clusterTestBundleTgz       = "/tmp/bundle.tgz"
	clusterTestClusterEndpoint = "https://cluster.local"
)

func TestBuildProfileReuseMode(t *testing.T) {
	opts := InstallOptions{
		ClusterEndpoint:  clusterTestClusterEndpoint,
		ValuesFile:       clusterTestValuesFile,
		ValuesPassphrase: clusterTestSecret,
	}
	profile, err := buildProfile(opts)
	if err != nil {
		t.Fatalf("buildProfile: %v", err)
	}
	if profile.Mode != config.ModeReuse {
		t.Fatalf("expected reuse mode, got %s", profile.Mode)
	}
	if profile.ClusterEndpoint != opts.ClusterEndpoint {
		t.Fatalf("expected cluster endpoint to be set, got %s", profile.ClusterEndpoint)
	}
}

func TestBuildProfileAirgappedRequiresBundle(t *testing.T) {
	opts := InstallOptions{
		Bootstrap:        true,
		ValuesFile:       clusterTestValuesFile,
		ValuesPassphrase: clusterTestSecret,
		Airgapped:        true,
	}
	if _, err := buildProfile(opts); err == nil {
		t.Fatalf("expected bundle path requirement error")
	}
	opts.BundlePath = clusterTestBundleTgz
	profile, err := buildProfile(opts)
	if err != nil {
		t.Fatalf("buildProfile with bundle: %v", err)
	}
	if !profile.Airgapped || profile.BundlePath == "" {
		t.Fatalf("expected airgapped profile with bundle path")
	}
}

func TestPrepareBundleSkipsWhenNotAirgapped(t *testing.T) {
	profile := &config.Profile{Airgapped: false}
	b, err := prepareBundle(profile, InstallOptions{}, InstallDeps{})
	if err != nil {
		t.Fatalf("prepareBundle: %v", err)
	}
	if b != nil {
		t.Fatalf("expected nil bundle when not airgapped")
	}
}

func TestEmitOutputUnsupportedFormat(t *testing.T) {
	cmd := &cobra.Command{}
	err := emitOutput(cmd, &config.Profile{}, &bundle.Bundle{}, false, "yaml")
	if err != errUnsupportedOutput {
		t.Fatalf("expected errUnsupportedOutput, got %v", err)
	}
}
