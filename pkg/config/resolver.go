package config

import (
	"errors"
	"fmt"
)

var (
	// ErrCommandNotDeclared indicates the requested command is absent from the declarative configuration.
	ErrCommandNotDeclared = errors.New("command not declared in configuration")
	// ErrCommandDisabled indicates the command was explicitly disabled in the declarative configuration.
	ErrCommandDisabled = errors.New("command disabled in configuration")
	// ErrUnknownProfile indicates a referenced profile name could not be found.
	ErrUnknownProfile = errors.New("profile not defined in configuration")
)

// ResolveInvocation merges defaults, reusable profiles, command-specific entries, and runtime overrides
// into a single flag set for the provided command path.
func ResolveInvocation(profile *ConfigurationProfile, commandPath string, runtime FlagSet) (*ResolvedInvocation, error) {
	section, err := validateCommandSection(profile, commandPath)
	if err != nil {
		return nil, err
	}

	resolved := newResolvedInvocation(commandPath, profile, section)
	applier := newFlagApplier(resolved)

	applier.Apply("defaults", profile.Defaults, ValueSourceDefault)

	if err := applyProfileSections(applier, profile, section); err != nil {
		return nil, err
	}

	applier.Apply("command "+commandPath, section.Flags, ValueSourceCommand)
	applier.Apply("runtime", sanitizeRuntimeOverrides(runtime), ValueSourceRuntime)

	return resolved, nil
}

func validateCommandSection(profile *ConfigurationProfile, commandPath string) (CommandSection, error) {
	if profile == nil {
		return CommandSection{}, ErrCommandNotDeclared
	}
	section, ok := profile.Commands[commandPath]
	if !ok {
		return CommandSection{}, ErrCommandNotDeclared
	}
	if section.Disabled {
		return CommandSection{}, ErrCommandDisabled
	}
	return section, nil
}

func newResolvedInvocation(commandPath string, profile *ConfigurationProfile, section CommandSection) *ResolvedInvocation {
	resolved := &ResolvedInvocation{
		CommandPath: commandPath,
		Flags:       FlagSet{},
		SourcePath:  profile.SourcePath,
	}
	if len(section.Profiles) > 0 {
		resolved.Profiles = append([]string(nil), section.Profiles...)
	}
	return resolved
}

type flagApplier struct {
	resolved *ResolvedInvocation
}

func newFlagApplier(resolved *ResolvedInvocation) *flagApplier {
	return &flagApplier{resolved: resolved}
}

func (a *flagApplier) Apply(source string, set FlagSet, fallback ValueSource) {
	if len(set) == 0 {
		return
	}
	for name, value := range set {
		current := value
		if current.Source == "" {
			current.Source = fallback
		}
		if previous, ok := a.resolved.Flags[name]; ok {
			a.resolved.Overrides = append(a.resolved.Overrides, fmt.Sprintf("%s overrides %s (was %s)", source, name, previous.Source))
		}
		a.resolved.Flags[name] = current
	}
}

func applyProfileSections(applier *flagApplier, profile *ConfigurationProfile, section CommandSection) error {
	if len(section.Profiles) == 0 {
		return nil
	}
	for _, name := range section.Profiles {
		flagSet, ok := profile.Profiles[name]
		if !ok {
			return fmt.Errorf("%w: %s", ErrUnknownProfile, name)
		}
		applier.Apply("profile "+name, flagSet, ValueSourceProfile)
	}
	return nil
}

func sanitizeRuntimeOverrides(runtime FlagSet) FlagSet {
	if len(runtime) == 0 {
		return nil
	}
	cloned := runtime.Clone()
	for name, value := range cloned {
		if value.Source == "" {
			value.Source = ValueSourceRuntime
			cloned[name] = value
		}
	}
	return cloned
}
