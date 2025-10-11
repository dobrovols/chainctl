package cluster

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/dobrovols/chainctl/cmd/chainctl/declarative"
	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/internal/validation"
	"github.com/dobrovols/chainctl/pkg/bootstrap"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/helm"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

// InstallOptions captures CLI flag values.
type InstallOptions struct {
	Bootstrap        bool
	ClusterEndpoint  string
	K3sVersion       string
	ValuesFile       string
	ValuesPassphrase string
	BundlePath       string
	Airgapped        bool
	DryRun           bool
	Output           string
}

// Bootstrapper performs k3s bootstrap when required.
type Bootstrapper interface {
	Bootstrap(*config.Profile) error
}

// HelmInstaller manages Helm install/upgrade flows.
type HelmInstaller interface {
	Install(*config.Profile, *bundle.Bundle) error
}

// InstallDeps configures dependencies for the install command.
type InstallDeps struct {
	Inspector           validation.SystemInspector
	BundleLoader        func(string, string) (*bundle.Bundle, error)
	Bootstrapper        Bootstrapper
	HelmInstaller       HelmInstaller
	TelemetryEmitter    func(io.Writer) (*telemetry.Emitter, error)
	ClusterValidator    func(*rest.Config) error
	ClusterConfigLoader func(*config.Profile) (*rest.Config, error)
}

var (
	errValuesFileRequired = errors.New("values file path is required")
	errUnsupportedOutput  = errors.New("unsupported output format")
	errBundleRequired     = errors.New("bundle path required when air-gapped")
)

// ErrBundleRequired exposes the sentinel.
func ErrBundleRequired() error { return errBundleRequired }

// ErrValuesFileRequired exposes the sentinel.
func ErrValuesFileRequired() error { return errValuesFileRequired }

// ErrUnsupportedOutput exposes the sentinel.
func ErrUnsupportedOutput() error { return errUnsupportedOutput }

// defaultInstallDeps used in production.
var defaultInstallDeps = InstallDeps{
	Inspector:           validation.DefaultInspector{},
	BundleLoader:        bundle.Load,
	Bootstrapper:        noopBootstrap{},
	HelmInstaller:       noopHelm{},
	TelemetryEmitter:    telemetry.NewEmitter,
	ClusterValidator:    validation.ValidateCluster,
	ClusterConfigLoader: loadClusterConfig,
}

type noopBootstrap struct{}

func (noopBootstrap) Bootstrap(*config.Profile) error { return nil }

type noopHelm struct{}

func (noopHelm) Install(*config.Profile, *bundle.Bundle) error { return nil }

// NewInstallCommand constructs the `chainctl cluster install` command.
func NewInstallCommand() *cobra.Command {
	opts := InstallOptions{}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install or upgrade the micro-services application",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runInstall(cmd, opts, defaultInstallDeps)
		},
	}

	cmd.Flags().BoolVar(&opts.Bootstrap, "bootstrap", false, "Bootstrap a new k3s cluster before install")
	cmd.Flags().StringVar(&opts.ClusterEndpoint, "cluster-endpoint", "", "Existing cluster API endpoint (reuse mode)")
	cmd.Flags().StringVar(&opts.K3sVersion, "k3s-version", "", "Target k3s version for bootstrap/upgrade")
	cmd.Flags().StringVar(&opts.ValuesFile, "values-file", "", "Encrypted Helm values file path")
	cmd.Flags().StringVar(&opts.ValuesPassphrase, "values-passphrase", "", "Passphrase for encrypted values")
	cmd.Flags().StringVar(&opts.BundlePath, "bundle-path", "", "Mounted bundle path when air-gapped")
	cmd.Flags().BoolVar(&opts.Airgapped, "airgapped", false, "Use air-gapped mode (requires --bundle-path)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Run validations without applying changes")
	cmd.Flags().StringVar(&opts.Output, "output", "text", "Output format: text or json")
	markDeclarative(cmd)

	return cmd
}

// RunInstallForTest executes the install flow with explicit dependencies (used in tests).
func RunInstallForTest(cmd *cobra.Command, opts InstallOptions, deps InstallDeps) error {
	return runInstall(cmd, opts, deps)
}

func runInstall(cmd *cobra.Command, opts InstallOptions, deps InstallDeps) (err error) {
	if strings.TrimSpace(opts.ValuesFile) == "" {
		return errValuesFileRequired
	}

	profile, err := buildProfile(opts)
	if err != nil {
		return err
	}

	if opts.Airgapped && strings.TrimSpace(opts.BundlePath) == "" {
		return errBundleRequired
	}

	inspector := deps.Inspector
	if inspector == nil {
		inspector = validation.DefaultInspector{}
	}

	hostResult := validation.ValidateHost(validation.HostConfig{
		RequireSudo:   true,
		MinCPU:        2,
		MinMemoryGiB:  4,
		KernelModules: []string{"br_netfilter", "overlay"},
	}, inspector)

	if !hostResult.Passed {
		return fmt.Errorf("preflight failed: %s", strings.Join(hostResult.Issues, "; "))
	}

	emitter := deps.TelemetryEmitter
	if emitter == nil {
		emitter = telemetry.NewEmitter
	}
	tel, err := emitter(cmd.OutOrStdout())
	if err != nil {
		return fmt.Errorf("initialize structured logging: %w", err)
	}
	logger := tel.StructuredLogger()
	if logger == nil {
		return fmt.Errorf("structured logger unavailable")
	}
	if resolved, ok := declarative.ResolvedInvocationFromContext(cmd); ok {
		declarative.EmitTelemetry(logger, resolved)
	}
	bootstrapHasLogging := false
	if orch, ok := deps.Bootstrapper.(*bootstrap.Orchestrator); ok {
		orch.WithLogging(logger)
		bootstrapHasLogging = true
	}
	helmHasLogging := false
	if installer, ok := deps.HelmInstaller.(*helm.Installer); ok {
		deps.HelmInstaller = installer.WithLogger(logger)
		helmHasLogging = true
	}

	commandMetadata := map[string]string{
		"mode":      string(profile.Mode),
		"namespace": profile.HelmNamespace,
		"release":   profile.HelmRelease,
	}
	if profile.Airgapped {
		commandMetadata["bundlePath"] = profile.BundlePath
	}
	logWorkflowStart(logger, stepInstall, commandMetadata)
	defer func() {
		if err != nil {
			logWorkflowFailure(logger, stepInstall, commandMetadata, err)
		}
	}()

	if profile.Mode == config.ModeReuse {
		loader := deps.ClusterConfigLoader
		if loader == nil {
			loader = loadClusterConfig
		}
		cfg, err := loader(profile)
		if err != nil {
			return fmt.Errorf("load cluster config: %w", err)
		}
		if cfg != nil {
			validator := deps.ClusterValidator
			if validator == nil {
				validator = validation.ValidateCluster
			}
			if err := validator(cfg); err != nil {
				return fmt.Errorf("cluster validation failed: %w", err)
			}
		}
	}

	bundleInstance, err := prepareBundle(profile, opts, deps)
	if err != nil {
		return err
	}

	helmArgsDryRun := buildHelmCommandArgs(profile, opts, true)
	if opts.DryRun {
		if profile.Mode == config.ModeBootstrap && !bootstrapHasLogging {
			logCommandEntry(logger, stepBootstrap, buildBootstrapCommandArgs(profile), "", telemetry.SeverityInfo, commandMetadata, nil)
		}
		if !helmHasLogging {
			logCommandEntry(logger, stepHelm, helmArgsDryRun, "", telemetry.SeverityInfo, commandMetadata, nil)
		}
		logWorkflowSuccess(logger, stepInstall, commandMetadata)
		return emitOutput(cmd, profile, bundleInstance, true, opts.Output)
	}

	if err := tel.EmitPhase(telemetry.PhaseBootstrap, map[string]string{"mode": string(profile.Mode)}, func() error {
		if profile.Mode == config.ModeBootstrap {
			return deps.Bootstrapper.Bootstrap(profile)
		}
		return nil
	}); err != nil {
		if profile.Mode == config.ModeBootstrap && !bootstrapHasLogging {
			logCommandEntry(logger, stepBootstrap, buildBootstrapCommandArgs(profile), err.Error(), telemetry.SeverityError, commandMetadata, err)
		}
		return err
	}
	if profile.Mode == config.ModeBootstrap && !bootstrapHasLogging {
		logCommandEntry(logger, stepBootstrap, buildBootstrapCommandArgs(profile), "", telemetry.SeverityInfo, commandMetadata, nil)
	}

	helmArgs := buildHelmCommandArgs(profile, opts, false)
	if err := tel.EmitPhase(telemetry.PhaseHelm, map[string]string{"mode": string(profile.Mode)}, func() error {
		return deps.HelmInstaller.Install(profile, bundleInstance)
	}); err != nil {
		if !helmHasLogging {
			logCommandEntry(logger, stepHelm, helmArgs, err.Error(), telemetry.SeverityError, commandMetadata, err)
		}
		return err
	}
	if !helmHasLogging {
		logCommandEntry(logger, stepHelm, helmArgs, "", telemetry.SeverityInfo, commandMetadata, nil)
	}

	logWorkflowSuccess(logger, stepInstall, commandMetadata)
	return emitOutput(cmd, profile, bundleInstance, false, opts.Output)
}

func prepareBundle(profile *config.Profile, opts InstallOptions, deps InstallDeps) (*bundle.Bundle, error) {
	if !profile.Airgapped {
		return nil, nil
	}
	loader := deps.BundleLoader
	if loader == nil {
		loader = bundle.Load
	}
	cacheRoot := filepath.Dir(profile.BundlePath)
	return loader(profile.BundlePath, cacheRoot)
}

func buildProfile(opts InstallOptions) (*config.Profile, error) {
	mode := config.ModeBootstrap
	if !opts.Bootstrap || strings.TrimSpace(opts.ClusterEndpoint) != "" {
		mode = config.ModeReuse
	}

	loadOpts := config.LoadOptions{
		Mode:                mode,
		ClusterEndpoint:     opts.ClusterEndpoint,
		K3sVersion:          opts.K3sVersion,
		EncryptedValuesPath: opts.ValuesFile,
		ValuesPassphrase:    opts.ValuesPassphrase,
		AirgappedBundlePath: opts.BundlePath,
		Offline:             opts.Airgapped,
	}

	return loadOpts.Validate()
}

func loadClusterConfig(profile *config.Profile) (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
	cfg, err := clientConfig.ClientConfig()
	if err != nil {
		kubeconfigPath := loadingRules.GetDefaultFilename()
		return nil, fmt.Errorf("failed to load kubeconfig from %q: %w", kubeconfigPath, err)
	}
	return cfg, nil
}

func emitOutput(cmd *cobra.Command, profile *config.Profile, b *bundle.Bundle, dryRun bool, format string) error {
	switch format {
	case "text":
		status := "Installation completed successfully"
		if dryRun {
			status = "Dry-run completed; no changes applied"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s. Mode=%s Airgapped=%t\n", status, profile.Mode, profile.Airgapped)
		if b != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "Bundle version: %s\n", b.Manifest.Version)
		}
		return nil
	case "json":
		payload := map[string]interface{}{
			"mode":      profile.Mode,
			"airgapped": profile.Airgapped,
			"cluster":   profile.ClusterEndpoint,
			"dryRun":    dryRun,
			"status":    "success",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		if b != nil {
			payload["bundleVersion"] = b.Manifest.Version
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(payload)
	default:
		return errUnsupportedOutput
	}
}

func buildHelmCommandArgs(profile *config.Profile, opts InstallOptions, dryRun bool) []string {
	args := []string{"helm", "upgrade", profile.HelmRelease}
	if ns := strings.TrimSpace(profile.HelmNamespace); ns != "" {
		args = append(args, "--namespace", ns)
	}
	args = append(args, "--values", profile.EncryptedFile)
	if profile.Airgapped && strings.TrimSpace(opts.BundlePath) != "" {
		args = append(args, "--bundle-path", opts.BundlePath)
	}
	if strings.TrimSpace(profile.Passphrase) != "" {
		args = append(args, "--values-passphrase", profile.Passphrase)
	}
	if dryRun {
		args = append(args, "--dry-run")
	}
	return args
}

func buildBootstrapCommandArgs(profile *config.Profile) []string {
	args := []string{"k3s-install"}
	if strings.TrimSpace(profile.Passphrase) != "" {
		args = append(args, "--values-passphrase", profile.Passphrase)
	}
	return args
}
