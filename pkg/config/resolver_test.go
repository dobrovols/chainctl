package config_test

import (
	"errors"
	"testing"

	"github.com/dobrovols/chainctl/pkg/config"
)

func TestResolveInvocationMergesDefaultsCommandAndRuntime(t *testing.T) {
	profile := &config.ConfigurationProfile{
		Defaults: config.FlagSet{
			"namespace": {Value: "demo", Source: config.ValueSourceDefault},
		},
		Commands: map[string]config.CommandSection{
			"chainctl cluster install": {
				Flags: config.FlagSet{
					"chart": {Value: "oci://example/cluster:1.0.0", Source: config.ValueSourceCommand},
				},
			},
		},
		SourcePath: "/tmp/chainctl.yaml",
	}

	runtime := config.FlagSet{
		"namespace": {Value: "demo-override", Source: config.ValueSourceRuntime},
	}

	resolved, err := config.ResolveInvocation(profile, "chainctl cluster install", runtime)
	if err != nil {
		t.Fatalf("ResolveInvocation returned error: %v", err)
	}
	if resolved.Flags["namespace"].Value != "demo-override" {
		t.Fatalf("expected namespace override, got %v", resolved.Flags["namespace"].Value)
	}
	if resolved.Flags["namespace"].Source != config.ValueSourceRuntime {
		t.Fatalf("expected runtime source, got %s", resolved.Flags["namespace"].Source)
	}
	if resolved.Flags["chart"].Value != "oci://example/cluster:1.0.0" {
		t.Fatalf("expected chart value retained, got %v", resolved.Flags["chart"].Value)
	}
	if resolved.Flags["chart"].Source != config.ValueSourceCommand {
		t.Fatalf("expected chart source command, got %s", resolved.Flags["chart"].Source)
	}
	if len(resolved.Overrides) == 0 {
		t.Fatalf("expected overrides to record runtime precedence")
	}
}

func TestResolveInvocationProfilesApply(t *testing.T) {
	profile := &config.ConfigurationProfile{
		Defaults: config.FlagSet{
			"namespace": {Value: "demo", Source: config.ValueSourceDefault},
		},
		Profiles: map[string]config.FlagSet{
			"staging": {
				"namespace": {Value: "staging", Source: config.ValueSourceProfile},
			},
		},
		Commands: map[string]config.CommandSection{
			"chainctl app install": {
				Profiles: []string{"staging"},
				Flags: config.FlagSet{
					"chart": {Value: "oci://example/app:1.0.0", Source: config.ValueSourceCommand},
				},
			},
		},
	}

	resolved, err := config.ResolveInvocation(profile, "chainctl app install", nil)
	if err != nil {
		t.Fatalf("ResolveInvocation returned error: %v", err)
	}
	if resolved.Flags["namespace"].Value != "staging" {
		t.Fatalf("expected namespace from staging profile, got %v", resolved.Flags["namespace"].Value)
	}
	if resolved.Flags["namespace"].Source != config.ValueSourceProfile {
		t.Fatalf("expected profile source, got %s", resolved.Flags["namespace"].Source)
	}
}

func TestResolveInvocationDisabledCommand(t *testing.T) {
	profile := &config.ConfigurationProfile{
		Commands: map[string]config.CommandSection{
			"chainctl cluster install": {
				Disabled: true,
			},
		},
	}

	_, err := config.ResolveInvocation(profile, "chainctl cluster install", nil)
	if !errors.Is(err, config.ErrCommandDisabled) {
		t.Fatalf("expected ErrCommandDisabled, got %v", err)
	}
}

func TestResolveInvocationUnknownCommand(t *testing.T) {
	profile := &config.ConfigurationProfile{
		Commands: map[string]config.CommandSection{},
	}

	_, err := config.ResolveInvocation(profile, "chainctl cluster install", nil)
	if !errors.Is(err, config.ErrCommandNotDeclared) {
		t.Fatalf("expected ErrCommandNotDeclared, got %v", err)
	}
}
