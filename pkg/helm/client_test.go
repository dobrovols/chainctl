package helm_test

import (
	"errors"
	"testing"

	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/helm"
)

type fakeHelmExec struct {
	installed bool
	err       error
}

func (f *fakeHelmExec) UpgradeRelease(profile *config.Profile, b *bundle.Bundle) error {
	f.installed = true
	return f.err
}

func TestHelmInstallerSuccess(t *testing.T) {
	exec := &fakeHelmExec{}
	installer := helm.NewInstaller(exec)

	if err := installer.Install(&config.Profile{HelmRelease: "chainapp"}, &bundle.Bundle{}); err != nil {
		t.Fatalf("install: %v", err)
	}
	if !exec.installed {
		t.Fatalf("expected helm upgrade to run")
	}
}

func TestHelmInstallerPropagatesError(t *testing.T) {
	wantErr := errors.New("helm failed")
	exec := &fakeHelmExec{err: wantErr}
	installer := helm.NewInstaller(exec)

	err := installer.Install(&config.Profile{}, &bundle.Bundle{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected helm error, got %v", err)
	}
}
