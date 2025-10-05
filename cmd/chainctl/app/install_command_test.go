package app_test

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"

	appcmd "github.com/dobrovols/chainctl/cmd/chainctl/app"
)

func TestNewInstallCommandFlags(t *testing.T) {
	cmd := appcmd.NewInstallCommand()
	for _, name := range []string{"cluster-endpoint", "values-file", "values-passphrase", "bundle-path", "chart", "release-name", "app-version", "namespace", "state-file", "state-file-name", "output"} {
		if cmd.Flag(name) == nil {
			t.Fatalf("expected flag %s to exist", name)
		}
	}
}

func TestInstallCommandMutuallyExclusiveSources(t *testing.T) {
	deps := appcmd.InstallDeps{}
	opts := appcmd.InstallOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
		ChartReference:   "oci://example.com/app:1.0.0",
		BundlePath:       "/tmp/bundle.tar",
	}

	err := appcmd.RunInstallForTest(&cobra.Command{}, opts, deps)
	if !errors.Is(err, appcmd.ErrConflictingSources()) {
		t.Fatalf("expected conflicting sources error, got %v", err)
	}
}

func TestInstallCommandMissingSource(t *testing.T) {
	deps := appcmd.InstallDeps{}
	opts := appcmd.InstallOptions{
		ClusterEndpoint:  "https://cluster.local",
		ValuesFile:       "/tmp/values.enc",
		ValuesPassphrase: "secret",
	}

	err := appcmd.RunInstallForTest(&cobra.Command{}, opts, deps)
	if !errors.Is(err, appcmd.ErrMissingSource()) {
		t.Fatalf("expected missing source error, got %v", err)
	}
}
