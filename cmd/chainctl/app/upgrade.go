package app

import (
	"errors"
	"io"

	"github.com/spf13/cobra"

	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

// UpgradeOptions holds CLI flags for app upgrade.
type UpgradeOptions struct {
	ClusterEndpoint  string
	ValuesFile       string
	ValuesPassphrase string
	BundlePath       string
	ChartReference   string
	ReleaseName      string
	AppVersion       string
	Namespace        string
	StateFileName    string
	StateFilePath    string
	Output           string
	Airgapped        bool
}

// UpgradeDeps defines dependencies required by the upgrade command.
type UpgradeDeps struct {
	Installer        HelmInstaller
	BundleLoader     func(string, string) (*bundle.Bundle, error)
	TelemetryEmitter func(io.Writer) (*telemetry.Emitter, error)
	Resolver         ChartResolver
	StateManager     StateManager
}

var (
	errValuesFile         = errors.New("values file is required")
	errClusterEndpoint    = errors.New("cluster endpoint must be provided")
	errUnsupportedOutput  = errors.New("unsupported output format")
	errConflictingSources = errors.New("exactly one of --chart or --bundle-path must be provided")
	errMissingSource      = errors.New("a chart reference or bundle path must be provided")
)

// ErrValuesFileRequired exposes the sentinel.
func ErrValuesFileRequired() error { return errValuesFile }

// ErrClusterEndpointRequired exposes the sentinel.
func ErrClusterEndpointRequired() error { return errClusterEndpoint }

// ErrUnsupportedOutput exposes the sentinel.
func ErrUnsupportedOutput() error { return errUnsupportedOutput }

// ErrConflictingSources exposes the mutual exclusion sentinel.
func ErrConflictingSources() error { return errConflictingSources }

// ErrMissingSource exposes the missing source sentinel.
func ErrMissingSource() error { return errMissingSource }

// NewUpgradeCommand constructs the `chainctl app upgrade` command.
func NewUpgradeCommand() *cobra.Command {
	opts := UpgradeOptions{}
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade the micro-services application Helm release",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runAppAction(cmd, opts.shared(), defaultUpgradeDeps, actionUpgrade)
		},
	}

	bindCommonFlags(cmd, &opts)

	return cmd
}

// RunUpgradeForTest executes the upgrade flow with injected dependencies.
func RunUpgradeForTest(cmd *cobra.Command, opts UpgradeOptions, deps UpgradeDeps) error {
	cmd.SilenceUsage = true
	return runAppAction(cmd, opts.shared(), deps, actionUpgrade)
}
