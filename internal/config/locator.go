package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConfigSource identifies where the configuration file was discovered.
type ConfigSource string

const (
	ConfigSourceExplicit   ConfigSource = "explicit"
	ConfigSourceEnv        ConfigSource = "env"
	ConfigSourceWorkingDir ConfigSource = "working-dir"
	ConfigSourceXDG        ConfigSource = "xdg"
	ConfigSourceHome       ConfigSource = "home"
)

// LocationResult describes the discovered configuration file.
type LocationResult struct {
	Path   string
	Source ConfigSource
}

// ErrConfigNotFound is returned when no configuration file can be located.
var ErrConfigNotFound = errors.New("declarative configuration not found")

// LocateConfig discovers the declarative configuration file following the precedence rules:
// explicit path → CHAINCTL_CONFIG → ./chainctl.yaml → XDG config → ~/.config/chainctl/config.yaml.
func LocateConfig(explicitPath string) (LocationResult, error) {
	if path := strings.TrimSpace(explicitPath); path != "" {
		clean := filepath.Clean(path)
		abs, err := toAbsolute(clean)
		if err != nil {
			return LocationResult{}, err
		}
		if exists(abs) {
			return LocationResult{Path: abs, Source: ConfigSourceExplicit}, nil
		}
		return LocationResult{}, fmt.Errorf("%w: %s", ErrConfigNotFound, abs)
	}

	if path, ok := os.LookupEnv("CHAINCTL_CONFIG"); ok && strings.TrimSpace(path) != "" {
		abs, err := toAbsolute(path)
		if err != nil {
			return LocationResult{}, err
		}
		if exists(abs) {
			return LocationResult{Path: abs, Source: ConfigSourceEnv}, nil
		}
		return LocationResult{}, fmt.Errorf("%w: %s", ErrConfigNotFound, abs)
	}

	if wd, err := os.Getwd(); err == nil {
		path := filepath.Join(wd, "chainctl.yaml")
		if exists(path) {
			return LocationResult{Path: path, Source: ConfigSourceWorkingDir}, nil
		}
	}

	if xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); xdg != "" {
		path := filepath.Join(xdg, "chainctl", "config.yaml")
		if exists(path) {
			return LocationResult{Path: path, Source: ConfigSourceXDG}, nil
		}
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		path := filepath.Join(home, ".config", "chainctl", "config.yaml")
		if exists(path) {
			return LocationResult{Path: path, Source: ConfigSourceHome}, nil
		}
	}

	return LocationResult{}, ErrConfigNotFound
}

func toAbsolute(path string) (string, error) {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		path = filepath.Join(home, strings.TrimPrefix(path, "~"))
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("absolute path: %w", err)
	}
	return abs, nil
}

func exists(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}
