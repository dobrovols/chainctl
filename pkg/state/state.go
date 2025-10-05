package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ChartSource captures the origin of the Helm chart applied during install/update.
type ChartSource struct {
	Type      string `json:"type"`
	Reference string `json:"reference"`
	Digest    string `json:"digest,omitempty"`
}

// Record stores the last successful install or update metadata for the application.
type Record struct {
	Release         string      `json:"release"`
	Namespace       string      `json:"namespace"`
	Chart           ChartSource `json:"chart"`
	Version         string      `json:"version"`
	LastAction      string      `json:"lastAction"`
	Timestamp       string      `json:"timestamp"`
	ClusterEndpoint string      `json:"clusterEndpoint,omitempty"`
}

// Overrides defines user-supplied preferences for the state file location.
type Overrides struct {
	StateDirectory string
	StateFileName  string
	StateFilePath  string
}

// PathResolver resolves the effective filesystem path for the state file.
type PathResolver interface {
	Resolve(Overrides) (string, error)
}

// Manager coordinates persistence of application state records.
type Manager struct {
	resolver PathResolver
	dirPerm  os.FileMode
	filePerm os.FileMode
}

var (
	errPathResolverMissing = errors.New("state path resolver not configured")
	errEmptyStatePath      = errors.New("resolved state file path empty")
	errWriteFailed         = errors.New("state file could not be written")
)

// NewManager constructs a Manager with the provided resolver.
func NewManager(resolver PathResolver) *Manager {
	return &Manager{
		resolver: resolver,
		dirPerm:  0o700,
		filePerm: 0o600,
	}
}

// ErrWriteFailed exposes the write failure sentinel.
func ErrWriteFailed() error { return errWriteFailed }

func (m *Manager) resolvePath(overrides Overrides) (string, error) {
	if m == nil || m.resolver == nil {
		return "", errPathResolverMissing
	}
	path, err := m.resolver.Resolve(overrides)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", errEmptyStatePath
	}
	return path, nil
}

// Write persists the provided record to the resolved state path.
func (m *Manager) Write(record Record, overrides Overrides) (string, error) {
	path, err := m.resolvePath(overrides)
	if err != nil {
		return "", err
	}

	if record.Timestamp == "" {
		record.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	dir := filepath.Dir(path)
	created := false
	if _, statErr := os.Stat(dir); statErr != nil {
		if !errors.Is(statErr, os.ErrNotExist) {
			return "", fmt.Errorf("%w: %w", errWriteFailed, statErr)
		}
		if err := os.MkdirAll(dir, m.dirPerm); err != nil {
			return "", fmt.Errorf("%w: %w", errWriteFailed, err)
		}
		created = true
	}

	if created {
		if err := os.Chmod(dir, m.dirPerm); err != nil {
			return "", fmt.Errorf("%w: %w", errWriteFailed, err)
		}
	}

	tmp, err := os.CreateTemp(dir, "state-*.json")
	if err != nil {
		return "", fmt.Errorf("%w: %w", errWriteFailed, err)
	}
	defer os.Remove(tmp.Name())

	if err := tmp.Chmod(m.filePerm); err != nil {
		tmp.Close()
		return "", fmt.Errorf("%w: %w", errWriteFailed, err)
	}

	enc := json.NewEncoder(tmp)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(record); err != nil {
		tmp.Close()
		return "", fmt.Errorf("%w: %w", errWriteFailed, err)
	}

	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("%w: %w", errWriteFailed, err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", fmt.Errorf("%w: %w", errWriteFailed, err)
	}

	if err := os.Chmod(path, m.filePerm); err != nil {
		return "", fmt.Errorf("%w: %w", errWriteFailed, err)
	}

	return path, nil
}
