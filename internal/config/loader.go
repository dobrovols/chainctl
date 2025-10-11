package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	pkgconfig "github.com/dobrovols/chainctl/pkg/config"
)

var (
	// ErrSecretsDisallowed is returned when the configuration declares a sensitive flag value.
	ErrSecretsDisallowed = errors.New("secrets are not permitted in declarative configuration")
	// ErrUnknownCommand indicates the configuration references a command not recognised by the CLI.
	ErrUnknownCommand = errors.New("unknown command referenced in declarative configuration")
	// ErrUnknownFlag indicates the configuration references an unsupported flag.
	ErrUnknownFlag = errors.New("unknown flag referenced in declarative configuration")
	// ErrInvalidFlagType indicates a YAML value cannot be coerced to the expected flag type.
	ErrInvalidFlagType = errors.New("invalid flag value type")
)

// Loader parses declarative configuration files into strongly typed profiles.
type Loader struct {
	catalog FlagCatalog
}

// NewLoader constructs a Loader with the provided flag catalog.
func NewLoader(catalog FlagCatalog) *Loader {
	return &Loader{catalog: catalog}
}

// Load parses the YAML file at the supplied path, performing validation and returning a configuration profile.
func (l *Loader) Load(path string) (*pkgconfig.ConfigurationProfile, error) {
	raw, err := readRawProfile(path)
	if err != nil {
		return nil, err
	}

	profile := newConfigurationProfile(path, raw)

	if err := l.populateDefaults(profile, raw); err != nil {
		return nil, err
	}
	if err := l.populateProfiles(profile, raw); err != nil {
		return nil, err
	}
	if err := l.populateCommands(profile, raw); err != nil {
		return nil, err
	}

	return profile, nil
}

func readRawProfile(path string) (rawProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return rawProfile{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var raw rawProfile
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&raw); err != nil && err != io.EOF {
		return rawProfile{}, fmt.Errorf("parse declarative config %q: %w", path, err)
	}
	return raw, nil
}

func newConfigurationProfile(path string, raw rawProfile) *pkgconfig.ConfigurationProfile {
	return &pkgconfig.ConfigurationProfile{
		Metadata: pkgconfig.Metadata{
			Name:        raw.Metadata.Name,
			Description: raw.Metadata.Description,
		},
		Defaults:   pkgconfig.FlagSet{},
		Profiles:   map[string]pkgconfig.FlagSet{},
		Commands:   map[string]pkgconfig.CommandSection{},
		SourcePath: path,
	}
}

func (l *Loader) populateDefaults(profile *pkgconfig.ConfigurationProfile, raw rawProfile) error {
	if len(raw.Defaults) == 0 {
		return nil
	}
	defaults, err := l.buildFlagSet(raw.Defaults, "", pkgconfig.ValueSourceDefault)
	if err != nil {
		return err
	}
	profile.Defaults = defaults
	return nil
}

func (l *Loader) populateProfiles(profile *pkgconfig.ConfigurationProfile, raw rawProfile) error {
	for name, entries := range raw.Profiles {
		flagSet, err := l.buildFlagSet(entries, "", pkgconfig.ValueSourceProfile)
		if err != nil {
			return fmt.Errorf("profile %q: %w", name, err)
		}
		profile.Profiles[name] = flagSet
	}
	return nil
}

func (l *Loader) populateCommands(profile *pkgconfig.ConfigurationProfile, raw rawProfile) error {
	for command, section := range raw.Commands {
		cmdPath := strings.TrimSpace(command)
		if cmdPath == "" {
			continue
		}
		if err := l.ensureCommandSupported(cmdPath); err != nil {
			return err
		}
		flagSet, err := l.buildFlagSet(section.Flags, cmdPath, pkgconfig.ValueSourceCommand)
		if err != nil {
			return fmt.Errorf("command %q: %w", cmdPath, err)
		}
		profile.Commands[cmdPath] = pkgconfig.CommandSection{
			Profiles: append([]string(nil), section.Profiles...),
			Flags:    flagSet,
			Disabled: section.Disabled,
		}
	}
	return nil
}

func (l *Loader) ensureCommandSupported(cmdPath string) error {
	if l.catalog.IsCommandSupported(cmdPath) {
		return nil
	}
	available := l.catalog.Commands()
	return fmt.Errorf("%w: %s. Available commands: %s", ErrUnknownCommand, cmdPath, strings.Join(available, ", "))
}

type rawProfile struct {
	Metadata rawMetadata                  `yaml:"metadata"`
	Defaults map[string]any               `yaml:"defaults"`
	Profiles map[string]map[string]any    `yaml:"profiles"`
	Commands map[string]rawCommandSection `yaml:"commands"`
}

type rawMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type rawCommandSection struct {
	Profiles []string       `yaml:"profiles"`
	Flags    map[string]any `yaml:"flags"`
	Disabled bool           `yaml:"disabled"`
}

func (l *Loader) buildFlagSet(entries map[string]any, command string, fallback pkgconfig.ValueSource) (pkgconfig.FlagSet, error) {
	if len(entries) == 0 {
		return pkgconfig.FlagSet{}, nil
	}
	set := pkgconfig.FlagSet{}
	for name, raw := range entries {
		if isSensitive(name) {
			return nil, fmt.Errorf("%w: %s", ErrSecretsDisallowed, name)
		}

		var (
			flagType FlagType
			ok       bool
		)
		if command != "" {
			flagType, ok = l.catalog.FlagType(command, name)
		} else {
			flagType, ok = l.catalog.AnyFlagType(name)
		}
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownFlag, name)
		}

		value, err := coerceValue(name, raw, flagType)
		if err != nil {
			return nil, err
		}

		set[name] = pkgconfig.FlagValue{
			Value:  value,
			Source: fallback,
		}
	}
	return set, nil
}

func isSensitive(name string) bool {
	lower := strings.ToLower(name)
	sensitive := []string{"token", "secret", "passphrase", "password", "kubeconfig"}
	for _, marker := range sensitive {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func coerceValue(name string, raw any, flagType FlagType) (any, error) {
	switch flagType {
	case FlagTypeBool:
		return coerceBoolValue(name, raw)
	case FlagTypeStringSlice:
		return coerceStringSliceValue(name, raw)
	default:
		str, err := stringify(name, raw)
		if err != nil {
			return nil, err
		}
		return str, nil
	}
}

func coerceBoolValue(name string, raw any) (any, error) {
	switch v := raw.(type) {
	case bool:
		return v, nil
	case string:
		value, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return nil, fmt.Errorf("%w: %s expects boolean", ErrInvalidFlagType, name)
		}
		return value, nil
	default:
		return nil, fmt.Errorf("%w: %s expects boolean", ErrInvalidFlagType, name)
	}
}

func coerceStringSliceValue(name string, raw any) (any, error) {
	switch v := raw.(type) {
	case []interface{}:
		out := make([]string, len(v))
		for i, item := range v {
			str, err := stringify(name, item)
			if err != nil {
				return nil, err
			}
			out[i] = str
		}
		return out, nil
	case []string:
		return append([]string(nil), v...), nil
	case string:
		return []string{strings.TrimSpace(v)}, nil
	default:
		return nil, fmt.Errorf("%w: %s expects string list", ErrInvalidFlagType, name)
	}
}

func stringify(name string, value any) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case fmt.Stringer:
		return v.String(), nil
	case int, int64, float64, float32:
		return fmt.Sprint(v), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return "", fmt.Errorf("%w: %s expects string-compatible value", ErrInvalidFlagType, name)
	}
}
