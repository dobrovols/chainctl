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
	if profile == nil {
		return nil, ErrCommandNotDeclared
	}

	section, ok := profile.Commands[commandPath]
	if !ok {
		return nil, ErrCommandNotDeclared
	}
	if section.Disabled {
		return nil, ErrCommandDisabled
	}

	resolved := &ResolvedInvocation{
		CommandPath: commandPath,
		Flags:       FlagSet{},
		SourcePath:  profile.SourcePath,
	}
	if len(section.Profiles) > 0 {
		resolved.Profiles = append([]string(nil), section.Profiles...)
	}

	apply := func(source string, set FlagSet, fallback ValueSource) {
		if len(set) == 0 {
			return
		}
		for name, value := range set {
			current := value
			if current.Source == "" {
				current.Source = fallback
			}
			if previous, ok := resolved.Flags[name]; ok {
				resolved.Overrides = append(resolved.Overrides, fmt.Sprintf("%s overrides %s (was %s)", source, name, previous.Source))
			}
			resolved.Flags[name] = current
		}
	}

	if profile.Defaults != nil {
		apply("defaults", profile.Defaults, ValueSourceDefault)
	}

	if len(section.Profiles) > 0 {
		for _, name := range section.Profiles {
			flagSet, ok := profile.Profiles[name]
			if !ok {
				return nil, fmt.Errorf("%w: %s", ErrUnknownProfile, name)
			}
			apply("profile "+name, flagSet, ValueSourceProfile)
		}
	}

	if section.Flags != nil {
		apply("command "+commandPath, section.Flags, ValueSourceCommand)
	}

	if runtime != nil {
		runtimeSet := runtime.Clone()
		for name, fv := range runtimeSet {
			if fv.Source == "" {
				fv.Source = ValueSourceRuntime
			}
			runtimeSet[name] = fv
		}
		apply("runtime", runtimeSet, ValueSourceRuntime)
	}

	return resolved, nil
}
