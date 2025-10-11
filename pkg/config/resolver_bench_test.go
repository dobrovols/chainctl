package config_test

import (
	"testing"

	"github.com/dobrovols/chainctl/pkg/config"
)

func BenchmarkResolveInvocation(b *testing.B) {
	profile := &config.ConfigurationProfile{
		Defaults: config.FlagSet{
			"namespace":   {Value: "demo", Source: config.ValueSourceDefault},
			"values-file": {Value: "/etc/chainctl/values.enc", Source: config.ValueSourceDefault},
		},
		Profiles: map[string]config.FlagSet{
			"staging": {
				"namespace": {Value: "staging", Source: config.ValueSourceProfile},
			},
		},
		Commands: map[string]config.CommandSection{
			"chainctl cluster install": {
				Profiles: []string{"staging"},
				Flags: config.FlagSet{
					"output": {Value: "json", Source: config.ValueSourceCommand},
				},
			},
		},
	}

	runtime := config.FlagSet{
		"namespace": {Value: "runtime", Source: config.ValueSourceRuntime},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := config.ResolveInvocation(profile, "chainctl cluster install", runtime); err != nil {
			b.Fatalf("resolve invocation: %v", err)
		}
	}
}
