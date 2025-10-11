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
	if result, found, err := locateExplicit(explicitPath); err != nil || found {
		return result, err
	}
	if result, found, err := locateEnv(); err != nil || found {
		return result, err
	}

	locators := []func() (LocationResult, bool, error){
		locateWorkingDir,
		locateXDG,
		locateHome,
	}
	for _, locator := range locators {
		result, found, err := locator()
		if err != nil {
			return LocationResult{}, err
		}
		if found {
			return result, nil
		}
	}

	return LocationResult{}, ErrConfigNotFound
}

func locateExplicit(explicitPath string) (LocationResult, bool, error) {
	path := strings.TrimSpace(explicitPath)
	if path == "" {
		return LocationResult{}, false, nil
	}
	return resolveCandidate(path, ConfigSourceExplicit, true)
}

func locateEnv() (LocationResult, bool, error) {
	value, ok := os.LookupEnv("CHAINCTL_CONFIG")
	if !ok || strings.TrimSpace(value) == "" {
		return LocationResult{}, false, nil
	}
	return resolveCandidate(value, ConfigSourceEnv, true)
}

func locateWorkingDir() (LocationResult, bool, error) {
	wd, err := os.Getwd()
	if err != nil {
		return LocationResult{}, false, nil
	}
	candidate := filepath.Join(wd, "chainctl.yaml")
	return resolveCandidate(candidate, ConfigSourceWorkingDir, false)
}

func locateXDG() (LocationResult, bool, error) {
	root := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if root == "" {
		return LocationResult{}, false, nil
	}
	candidate := filepath.Join(root, "chainctl", "config.yaml")
	return resolveCandidate(candidate, ConfigSourceXDG, false)
}

func locateHome() (LocationResult, bool, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return LocationResult{}, false, nil
	}
	candidate := filepath.Join(home, ".config", "chainctl", "config.yaml")
	return resolveCandidate(candidate, ConfigSourceHome, false)
}

func resolveCandidate(path string, source ConfigSource, errorOnMissing bool) (LocationResult, bool, error) {
	clean := filepath.Clean(path)
	abs, err := toAbsolute(clean)
	if err != nil {
		return LocationResult{}, false, err
	}
	if exists(abs) {
		return LocationResult{Path: abs, Source: source}, true, nil
	}
	if errorOnMissing {
		return LocationResult{}, false, fmt.Errorf("%w: %s", ErrConfigNotFound, abs)
	}
	return LocationResult{}, false, nil
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
