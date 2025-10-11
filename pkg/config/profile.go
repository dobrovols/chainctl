package config

// ValueSource identifies where a flag value originated within the precedence chain.
type ValueSource string

const (
	// ValueSourceDefault indicates the value originated from the top-level defaults.
	ValueSourceDefault ValueSource = "default"
	// ValueSourceProfile indicates the value came from a reusable profile grouping.
	ValueSourceProfile ValueSource = "profile"
	// ValueSourceCommand indicates the value came from the command-specific section.
	ValueSourceCommand ValueSource = "command"
	// ValueSourceRuntime indicates the value was supplied directly at runtime.
	ValueSourceRuntime ValueSource = "runtime"
)

// FlagValue stores a typed flag value and its precedence origin.
type FlagValue struct {
	Value  any
	Source ValueSource
}

// FlagSet represents a mapping of flag names to values.
type FlagSet map[string]FlagValue

// Clone creates a deep copy of the flag set so future mutations do not affect the original.
func (f FlagSet) Clone() FlagSet {
	if len(f) == 0 {
		return FlagSet{}
	}
	out := make(FlagSet, len(f))
	for k, v := range f {
		out[k] = v
	}
	return out
}

// Metadata captures optional descriptive fields for a configuration profile.
type Metadata struct {
	Name        string
	Description string
}

// CommandSection represents the configuration for a specific command path.
type CommandSection struct {
	Profiles []string
	Flags    FlagSet
	Disabled bool
}

// Clone produces a deep copy of the command section.
func (c CommandSection) Clone() CommandSection {
	out := CommandSection{
		Disabled: c.Disabled,
	}
	if len(c.Profiles) > 0 {
		out.Profiles = append([]string(nil), c.Profiles...)
	}
	if len(c.Flags) > 0 {
		out.Flags = c.Flags.Clone()
	} else {
		out.Flags = FlagSet{}
	}
	return out
}

// ConfigurationProfile represents the parsed YAML configuration covering all supported commands.
type ConfigurationProfile struct {
	Metadata   Metadata
	Defaults   FlagSet
	Profiles   map[string]FlagSet
	Commands   map[string]CommandSection
	SourcePath string
}

// ResolvedInvocation captures the effective flag set for a single command after precedence resolution.
type ResolvedInvocation struct {
	CommandPath string
	Profiles    []string
	Flags       FlagSet
	Overrides   []string
	Warnings    []string
	SourcePath  string
}
