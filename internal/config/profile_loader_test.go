package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dobrovols/chainctl/internal/config"
)

func TestValidateProfileBootstrapDefaults(t *testing.T) {
	profile, err := (config.LoadOptions{
		Mode:                config.ModeBootstrap,
		EncryptedValuesPath: "/tmp/values.enc",
		ValuesPassphrase:    "secret",
		Offline:             true,
		AirgappedBundlePath: "/mnt/bundle",
	}).Validate()
	if err != nil {
		t.Fatalf("validate: %v", err)
	}

	if profile.Mode != config.ModeBootstrap {
		t.Fatalf("expected bootstrap mode, got %s", profile.Mode)
	}
	if !profile.Airgapped {
		t.Fatalf("expected airgapped true")
	}
	if profile.BundlePath != "/mnt/bundle" {
		t.Fatalf("expected cleaned bundle path, got %s", profile.BundlePath)
	}
	if profile.HelmNamespace != "chain-system" {
		t.Fatalf("expected default namespace, got %s", profile.HelmNamespace)
	}
	if profile.Passphrase != "secret" {
		t.Fatalf("passphrase not carried through")
	}
}

func TestValidateProfileReuseRequiresEndpoint(t *testing.T) {
	_, err := (config.LoadOptions{
		Mode:                config.ModeReuse,
		EncryptedValuesPath: "/tmp/values.enc",
	}).Validate()
	if !errors.Is(err, config.ErrClusterEndpointRequired()) {
		t.Fatalf("expected endpoint required error, got %v", err)
	}
}

func TestValidateProfileAirgappedRequiresBundle(t *testing.T) {
	_, err := (config.LoadOptions{
		Mode:                config.ModeBootstrap,
		EncryptedValuesPath: "/tmp/values.enc",
		Offline:             true,
	}).Validate()
	if !errors.Is(err, config.ErrBundlePathRequired()) {
		t.Fatalf("expected bundle path required error, got %v", err)
	}
}

func TestValidateProfileUnknownMode(t *testing.T) {
	_, err := (config.LoadOptions{
		Mode:                "invalid",
		EncryptedValuesPath: "/tmp/values.enc",
	}).Validate()
	if !errors.Is(err, config.ErrUnknownMode()) {
		t.Fatalf("expected unknown mode error, got %v", err)
	}
}

func TestValidateProfileReuseSuccess(t *testing.T) {
	profile, err := (config.LoadOptions{
		Mode:                config.ModeReuse,
		ClusterEndpoint:     "https://cluster.local",
		EncryptedValuesPath: "/tmp/values.enc",
		HelmReleaseName:     "custom",
	}).Validate()
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if profile.ClusterEndpoint != "https://cluster.local" {
		t.Fatalf("expected cluster endpoint to be set")
	}
	if profile.HelmRelease != "custom" {
		t.Fatalf("expected custom helm release, got %s", profile.HelmRelease)
	}
}

func TestErrEncryptedFileRequiredExposed(t *testing.T) {
	if config.ErrEncryptedFileRequired() == nil {
		t.Fatalf("expected non-nil error sentinel")
	}
}

func TestProfileStringMasksPassphrase(t *testing.T) {
	profile := &config.Profile{
		Mode:            config.ModeBootstrap,
		ClusterEndpoint: "https://cluster.local",
		Airgapped:       true,
		BundlePath:      "/mnt/bundle",
		EncryptedFile:   "/tmp/values.enc",
		Passphrase:      "secret",
	}

	summary := profile.String()
	if strings.Contains(summary, "secret") {
		t.Fatalf("expected passphrase to be masked, got %s", summary)
	}
	if !strings.Contains(summary, "passphrase=******") {
		t.Fatalf("expected masked token in summary, got %s", summary)
	}

	profile.Passphrase = ""
	summary = profile.String()
	if !strings.Contains(summary, "passphrase=<none>") {
		t.Fatalf("expected <none> marker when passphrase empty, got %s", summary)
	}
}
