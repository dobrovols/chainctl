package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Mode represents how the installer should operate.
type Mode string

const (
	ModeBootstrap Mode = "bootstrap"
	ModeReuse     Mode = "reuse"
)

// LoadOptions capture raw CLI inputs prior to validation.
type LoadOptions struct {
	Mode                Mode
	ClusterEndpoint     string
	AirgappedBundlePath string
	EncryptedValuesPath string
	ValuesPassphrase    string
	K3sVersion          string
	HelmReleaseName     string
	HelmNamespace       string
	Offline             bool
}

// Profile is the validated configuration used by the installer.
type Profile struct {
	Mode            Mode
	ClusterEndpoint string
	Airgapped       bool
	BundlePath      string
	EncryptedFile   string
	Passphrase      string
	K3sVersion      string
	HelmRelease     string
	HelmNamespace   string
}

var (
	errUnknownMode        = errors.New("unknown mode")
	errClusterEndpointReq = errors.New("cluster endpoint required for reuse mode")
	errEncryptedFileReq   = errors.New("encrypted values file path required")
	errBundlePathReq      = errors.New("bundle path required for air-gapped execution")
)

// ErrUnknownMode exposes the sentinel.
func ErrUnknownMode() error { return errUnknownMode }

// ErrClusterEndpointRequired exposes the sentinel.
func ErrClusterEndpointRequired() error { return errClusterEndpointReq }

// ErrEncryptedFileRequired exposes the sentinel.
func ErrEncryptedFileRequired() error { return errEncryptedFileReq }

// ErrBundlePathRequired exposes the sentinel.
func ErrBundlePathRequired() error { return errBundlePathReq }

// Validate converts options into a strongly-typed profile.
func (o LoadOptions) Validate() (*Profile, error) {
	mode := strings.ToLower(string(o.Mode))
	switch mode {
	case "", string(ModeBootstrap):
		o.Mode = ModeBootstrap
	case string(ModeReuse):
		o.Mode = ModeReuse
	default:
		return nil, errUnknownMode
	}

	if o.Mode == ModeReuse && strings.TrimSpace(o.ClusterEndpoint) == "" {
		return nil, errClusterEndpointReq
	}

	if strings.TrimSpace(o.EncryptedValuesPath) == "" {
		return nil, errEncryptedFileReq
	}

	profile := &Profile{
		Mode:          o.Mode,
		Airgapped:     o.Offline,
		EncryptedFile: filepath.Clean(o.EncryptedValuesPath),
		Passphrase:    o.ValuesPassphrase,
		K3sVersion:    o.K3sVersion,
		HelmRelease:   defaultString(o.HelmReleaseName, "chainapp"),
		HelmNamespace: defaultString(o.HelmNamespace, "chain-system"),
	}

	if o.Mode == ModeReuse {
		profile.ClusterEndpoint = o.ClusterEndpoint
	}

	if profile.Airgapped {
		if strings.TrimSpace(o.AirgappedBundlePath) == "" {
			return nil, errBundlePathReq
		}
		profile.BundlePath = filepath.Clean(o.AirgappedBundlePath)
	}

	return profile, nil
}

func defaultString(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}

// String returns a redacted summary for logging.
func (p *Profile) String() string {
	masked := "******"
	pass := masked
	if p.Passphrase == "" {
		pass = "<none>"
	}
	return fmt.Sprintf("mode=%s endpoint=%s airgapped=%t bundle=%s encrypted=%s passphrase=%s", p.Mode, p.ClusterEndpoint, p.Airgapped, p.BundlePath, p.EncryptedFile, pass)
}
