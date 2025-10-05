package state

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pkgstate "github.com/dobrovols/chainctl/pkg/state"
)

const (
	defaultFileName = "app.json"
	configDirName   = "chainctl"
	stateDirName    = "state"
)

var (
	errConflictingOverrides = errors.New("state file override is invalid: specify either --state-file or --state-file-name")
	errRelativeStateFile    = errors.New("state file override is invalid: must provide an absolute path")
	errInvalidFileName      = errors.New("state file override is invalid: filename must not contain path separators")
)

// Resolver resolves state file paths according to overrides and platform defaults.
type Resolver struct{}

// NewResolver constructs a state path resolver.
func NewResolver() *Resolver {
	return &Resolver{}
}

func (r *Resolver) Resolve(overrides pkgstate.Overrides) (string, error) {
	if overrides.StateFilePath != "" && overrides.StateFileName != "" {
		return "", errConflictingOverrides
	}

	if overrides.StateFilePath != "" {
		if !filepath.IsAbs(overrides.StateFilePath) {
			return "", errRelativeStateFile
		}
		return filepath.Clean(overrides.StateFilePath), nil
	}

	dir := overrides.StateDirectory
	if dir == "" {
		var err error
		dir, err = defaultStateDirectory()
		if err != nil {
			return "", fmt.Errorf("determine state directory: %w", err)
		}
	}

	dir = filepath.Clean(dir)
	if !filepath.IsAbs(dir) {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("resolve state directory: %w", err)
		}
		dir = abs
	}

	fileName := overrides.StateFileName
	if fileName == "" {
		fileName = defaultFileName
	}
	if invalidFileName(fileName) {
		return "", errInvalidFileName
	}

	return filepath.Join(dir, fileName), nil
}

func invalidFileName(name string) bool {
	if name == "" || strings.ContainsAny(name, `/\`) {
		return true
	}
	// Check for control characters
	for _, r := range name {
		if r < 32 || r == 127 {
			return true
		}
	}
	// Reserved Windows filenames (case-insensitive)
	reserved := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}
	upper := strings.ToUpper(name)
	for _, res := range reserved {
		if upper == res || strings.HasPrefix(upper, res+".") {
			return true
		}
	}
	return false
}

func defaultStateDirectory() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(filepath.Clean(xdg), configDirName, stateDirName), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", errors.New("unable to determine user home directory")
	}

	return filepath.Join(filepath.Clean(home), "."+configDirName, stateDirName), nil
}

// ErrConflictingOverrides exposes the override validation error.
func ErrConflictingOverrides() error { return errConflictingOverrides }

// ErrRelativeStateFile exposes the relative path validation error.
func ErrRelativeStateFile() error { return errRelativeStateFile }

// ErrInvalidFileName exposes invalid filename validation error.
func ErrInvalidFileName() error { return errInvalidFileName }
