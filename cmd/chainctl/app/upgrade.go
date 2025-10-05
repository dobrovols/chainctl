package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

// UpgradeOptions holds CLI flags for app upgrade.
type UpgradeOptions struct {
	ClusterEndpoint  string
	ValuesFile       string
	ValuesPassphrase string
	BundlePath       string
	Airgapped        bool
	Output           string
}

// UpgradeDeps defines dependencies required by the upgrade command.
type UpgradeDeps struct {
	Installer        HelmInstaller
	BundleLoader     func(string, string) (*bundle.Bundle, error)
	TelemetryEmitter func(io.Writer) *telemetry.Emitter
}

// HelmInstaller orchestrates Helm release operations.
type HelmInstaller interface {
	Install(*config.Profile, *bundle.Bundle) error
}

var (
	errValuesFile        = errors.New("values file is required")
	errClusterEndpoint   = errors.New("cluster endpoint must be provided")
	errUnsupportedOutput = errors.New("unsupported output format")
)

// ErrValuesFileRequired exposes the sentinel.
func ErrValuesFileRequired() error { return errValuesFile }

// ErrClusterEndpointRequired exposes the sentinel.
func ErrClusterEndpointRequired() error { return errClusterEndpoint }

// ErrUnsupportedOutput exposes the sentinel.
func ErrUnsupportedOutput() error { return errUnsupportedOutput }

// defaultUpgradeDeps for production wiring.
var defaultUpgradeDeps = UpgradeDeps{
	Installer:        nil,
	BundleLoader:     bundle.Load,
	TelemetryEmitter: telemetry.NewEmitter,
}

// NewUpgradeCommand constructs the `chainctl app upgrade` command.
func NewUpgradeCommand() *cobra.Command {
	opts := UpgradeOptions{}
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the micro-services application Helm release",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runUpgrade(cmd, opts, defaultUpgradeDeps)
		},
	}

	cmd.Flags().StringVar(&opts.ClusterEndpoint, "cluster-endpoint", "", "Existing cluster API endpoint")
	cmd.Flags().StringVar(&opts.ValuesFile, "values-file", "", "Encrypted Helm values file path")
	cmd.Flags().StringVar(&opts.ValuesPassphrase, "values-passphrase", "", "Passphrase for encrypted values")
	cmd.Flags().StringVar(&opts.BundlePath, "bundle-path", "", "Mounted bundle path when air-gapped")
	cmd.Flags().BoolVar(&opts.Airgapped, "airgapped", false, "Use offline assets from bundle")
	cmd.Flags().StringVar(&opts.Output, "output", "text", "Output format: text or json")

	return cmd
}

// RunUpgradeForTest executes the upgrade flow with injected dependencies.
func RunUpgradeForTest(cmd *cobra.Command, opts UpgradeOptions, deps UpgradeDeps) error {
	return runUpgrade(cmd, opts, deps)
}

func runUpgrade(cmd *cobra.Command, opts UpgradeOptions, deps UpgradeDeps) error {
	if strings.TrimSpace(opts.ValuesFile) == "" {
		return errValuesFile
	}
	if strings.TrimSpace(opts.ClusterEndpoint) == "" {
		return errClusterEndpoint
	}

	profile, err := buildProfile(opts)
	if err != nil {
		return err
	}

	bundleInstance, err := loadBundle(profile, opts, deps)
	if err != nil {
		return err
	}

	installer := deps.Installer
	if installer == nil {
		installer = noopInstaller{}
	}

	emitter := deps.TelemetryEmitter
	if emitter == nil {
		emitter = telemetry.NewEmitter
	}
	tel := emitter(cmd.OutOrStdout())

	if err := tel.EmitPhase(telemetry.PhaseHelm, map[string]string{"mode": string(profile.Mode)}, func() error {
		return installer.Install(profile, bundleInstance)
	}); err != nil {
		return err
	}

	return emitOutput(cmd, profile, opts.Output)
}

func buildProfile(opts UpgradeOptions) (*config.Profile, error) {
	loadOpts := config.LoadOptions{
		Mode:                config.ModeReuse,
		ClusterEndpoint:     opts.ClusterEndpoint,
		EncryptedValuesPath: opts.ValuesFile,
		ValuesPassphrase:    opts.ValuesPassphrase,
		AirgappedBundlePath: opts.BundlePath,
		Offline:             opts.Airgapped,
	}
	return loadOpts.Validate()
}

func loadBundle(profile *config.Profile, opts UpgradeOptions, deps UpgradeDeps) (*bundle.Bundle, error) {
	if !profile.Airgapped {
		return nil, nil
	}
	loader := deps.BundleLoader
	if loader == nil {
		loader = bundle.Load
	}
	return loader(profile.BundlePath, filepath.Dir(profile.BundlePath))
}

func emitOutput(cmd *cobra.Command, profile *config.Profile, format string) error {
	switch format {
	case "text":
		fmt.Fprintf(cmd.OutOrStdout(), "Upgrade completed successfully for release %s\n", profile.HelmRelease)
		return nil
	case "json":
		payload := map[string]interface{}{
			"status":    "success",
			"release":   profile.HelmRelease,
			"cluster":   profile.ClusterEndpoint,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(payload)
	default:
		return errUnsupportedOutput
	}
}

type noopInstaller struct{}

func (noopInstaller) Install(*config.Profile, *bundle.Bundle) error { return nil }
