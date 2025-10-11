package config_test

import (
	"testing"

	"github.com/dobrovols/chainctl/pkg/config"
)

func TestFlagSetCloneIsIndependent(t *testing.T) {
	source := config.FlagSet{
		"namespace": {Value: "demo", Source: config.ValueSourceDefault},
	}

	cloned := source.Clone()
	cloned["namespace"] = config.FlagValue{Value: "other", Source: config.ValueSourceCommand}

	if source["namespace"].Value != "demo" {
		t.Fatalf("expected original flag set to remain unchanged, got %v", source["namespace"].Value)
	}
}

func TestCommandSectionCloneCopiesProfilesAndFlags(t *testing.T) {
	section := config.CommandSection{
		Profiles: []string{"staging"},
		Flags: config.FlagSet{
			"chart": {Value: "oci://example/app:1.0.0", Source: config.ValueSourceCommand},
		},
	}

	cloned := section.Clone()
	cloned.Profiles[0] = "prod"
	cloned.Flags["chart"] = config.FlagValue{Value: "oci://example/app:2.0.0", Source: config.ValueSourceCommand}

	if section.Profiles[0] != "staging" {
		t.Fatalf("expected original profiles slice to remain unchanged, got %s", section.Profiles[0])
	}
	if section.Flags["chart"].Value != "oci://example/app:1.0.0" {
		t.Fatalf("expected original flag value unchanged, got %v", section.Flags["chart"].Value)
	}
}
