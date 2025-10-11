package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/dobrovols/chainctl/cmd/chainctl/declarative"
	"github.com/dobrovols/chainctl/internal/config"
	internalstate "github.com/dobrovols/chainctl/internal/state"
	"github.com/dobrovols/chainctl/pkg/bundle"
	"github.com/dobrovols/chainctl/pkg/helm"
	pkgstate "github.com/dobrovols/chainctl/pkg/state"
	"github.com/dobrovols/chainctl/pkg/telemetry"
)

type appAction string

const (
	actionInstall appAction = "install"
	actionUpgrade appAction = "upgrade"
)

type sharedOptions struct {
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
}

type ChartResolver interface {
	Resolve(ctx context.Context, opts helm.ResolveOptions) (helm.ResolveResult, error)
}

// StateManager persists application execution state.
type StateManager interface {
	Write(pkgstate.Record, pkgstate.Overrides) (string, error)
}

type resolutionResult struct {
	Outcome helm.ResolveResult
	Bundle  *bundle.Bundle
}

var (
	errResolverPullerMissing = errors.New("oci puller not configured")
	titleCaser               = cases.Title(language.English)
)

func (o UpgradeOptions) shared() sharedOptions {
	return sharedOptions{
		ClusterEndpoint:  o.ClusterEndpoint,
		ValuesFile:       o.ValuesFile,
		ValuesPassphrase: o.ValuesPassphrase,
		BundlePath:       o.BundlePath,
		ChartReference:   o.ChartReference,
		ReleaseName:      o.ReleaseName,
		AppVersion:       o.AppVersion,
		Namespace:        o.Namespace,
		StateFileName:    o.StateFileName,
		StateFilePath:    o.StateFilePath,
		Output:           o.Output,
	}
}

func runAppAction(cmd *cobra.Command, options sharedOptions, deps UpgradeDeps, action appAction) (err error) {
	ensureDeps(&deps)

	if err := validateAppActionInputs(options, action); err != nil {
		return err
	}

	profile, err := buildProfileForAction(options, action)
	if err != nil {
		return err
	}

	stateOverrides, statePathHint, err := resolveStateOverrides(options)
	if err != nil {
		return err
	}

	tel, logger, err := initAppTelemetry(cmd, deps)
	if err != nil {
		return err
	}

	workflowStep := workflowStepName(action)
	workflowMetadata := buildWorkflowMetadata(profile)
	logWorkflowStart(logger, workflowStep, workflowMetadata)
	defer func() {
		if err != nil {
			logWorkflowFailure(logger, workflowStep, workflowMetadata, err)
		}
	}()

	resolved, err := resolveChartWithLogging(cmd.Context(), options, deps, logger, workflowMetadata)
	if err != nil {
		return err
	}

	bundleInstance := resolved.Bundle
	installer, helmHasLogging := prepareAppInstaller(deps.Installer, logger)

	helmMetadata := buildHelmInstallMetadata(profile, resolved.Outcome)
	helmArgs := buildHelmInstallArgs(profile, options)

	if err = executeHelmPhase(tel, installer, profile, bundleInstance, helmMetadata, helmArgs, logger, helmHasLogging); err != nil {
		return err
	}

	record := buildStateRecord(profile, resolved.Outcome, action, options)
	statePath, err := persistState(deps.StateManager, record, stateOverrides, statePathHint)
	if err != nil {
		return err
	}

	logWorkflowSuccess(logger, workflowStep, workflowMetadata)
	return emitOutput(cmd, profile, resolved.Outcome, statePath, options.Output, action, options)
}

func validateAppActionInputs(options sharedOptions, action appAction) error {
	if strings.TrimSpace(options.ValuesFile) == "" {
		return errValuesFile
	}
	if action == actionUpgrade && strings.TrimSpace(options.ClusterEndpoint) == "" {
		return errClusterEndpoint
	}
	return nil
}

func initAppTelemetry(cmd *cobra.Command, deps UpgradeDeps) (*telemetry.Emitter, telemetry.StructuredLogger, error) {
	emitter := deps.TelemetryEmitter
	if emitter == nil {
		emitter = telemetryEmitterDefault
	}

	tel, err := emitter(cmd.ErrOrStderr())
	if err != nil {
		return nil, nil, fmt.Errorf("initialize structured logging: %w", err)
	}

	logger := tel.StructuredLogger()
	if logger == nil {
		return nil, nil, fmt.Errorf("structured logger unavailable")
	}

	if resolved, ok := declarative.ResolvedInvocationFromContext(cmd); ok {
		declarative.EmitTelemetry(logger, resolved)
	}

	return tel, logger, nil
}

func buildWorkflowMetadata(profile *config.Profile) map[string]string {
	metadata := map[string]string{
		"mode":      string(profile.Mode),
		"namespace": profile.HelmNamespace,
		"release":   profile.HelmRelease,
	}
	if profile.ClusterEndpoint != "" {
		metadata["cluster"] = profile.ClusterEndpoint
	}
	return metadata
}

func resolveChartWithLogging(
	ctx context.Context,
	options sharedOptions,
	deps UpgradeDeps,
	logger telemetry.StructuredLogger,
	workflowMetadata map[string]string,
) (resolutionResult, error) {
	resolved, err := resolveChartSource(ctx, options, deps)
	resolveArgs := buildResolveArgs(options)
	if err != nil {
		resolveMetadata := buildResolveMetadata(nil, options)
		logCommandEntry(logger, stepHelmResolve, resolveArgs, err.Error(), telemetry.SeverityError, resolveMetadata, err)
		return resolutionResult{}, err
	}

	resolveMetadata := buildResolveMetadata(&resolved.Outcome, options)
	logCommandEntry(logger, stepHelmResolve, resolveArgs, "", telemetry.SeverityInfo, resolveMetadata, nil)
	updateWorkflowMetadataFromOutcome(workflowMetadata, resolved.Outcome)

	return resolved, nil
}

func updateWorkflowMetadataFromOutcome(metadata map[string]string, outcome helm.ResolveResult) {
	if outcome.Source.Type != "" {
		metadata["source"] = outcome.Source.Type
	}
	if outcome.Source.Digest != "" {
		metadata["digest"] = outcome.Source.Digest
	}
}

func prepareAppInstaller(installer HelmInstaller, logger telemetry.StructuredLogger) (HelmInstaller, bool) {
	if installer == nil {
		return noopInstaller{}, false
	}
	if typed, ok := installer.(*helm.Installer); ok {
		return typed.WithLogger(logger), true
	}
	return installer, false
}

func executeHelmPhase(
	tel *telemetry.Emitter,
	installer HelmInstaller,
	profile *config.Profile,
	bundleInstance *bundle.Bundle,
	helmMetadata map[string]string,
	helmArgs []string,
	logger telemetry.StructuredLogger,
	helmHasLogging bool,
) error {
	execute := func() error {
		installErr := installer.Install(profile, bundleInstance)
		if !helmHasLogging {
			stderr := ""
			severity := telemetry.SeverityInfo
			if installErr != nil {
				severity = telemetry.SeverityError
				stderr = installErr.Error()
			}
			logCommandEntry(logger, stepHelmCommand, helmArgs, stderr, severity, helmMetadata, installErr)
		}
		return installErr
	}

	return tel.EmitPhase(telemetry.PhaseHelm, helmMetadata, execute)
}

func buildStateRecord(profile *config.Profile, outcome helm.ResolveResult, action appAction, opts sharedOptions) pkgstate.Record {
	return pkgstate.Record{
		Release:         profile.HelmRelease,
		Namespace:       profile.HelmNamespace,
		Chart:           outcome.Source,
		Version:         deriveVersion(opts, outcome),
		LastAction:      string(action),
		ClusterEndpoint: profile.ClusterEndpoint,
	}
}

func persistState(manager StateManager, record pkgstate.Record, overrides pkgstate.Overrides, hint string) (string, error) {
	if manager == nil {
		return "", fmt.Errorf("state manager unavailable")
	}

	statePath, err := manager.Write(record, overrides)
	if err != nil {
		return "", fmt.Errorf("state file could not be written: %w", err)
	}
	if statePath == "" {
		statePath = hint
	}
	return statePath, nil
}

func buildProfileForAction(opts sharedOptions, action appAction) (*config.Profile, error) {
	load := config.LoadOptions{
		Mode:                config.ModeReuse,
		ClusterEndpoint:     opts.ClusterEndpoint,
		EncryptedValuesPath: opts.ValuesFile,
		ValuesPassphrase:    opts.ValuesPassphrase,
		AirgappedBundlePath: opts.BundlePath,
		Offline:             strings.TrimSpace(opts.BundlePath) != "",
		HelmReleaseName:     opts.ReleaseName,
		HelmNamespace:       opts.Namespace,
	}

	if action == actionInstall {
		load.Mode = config.ModeBootstrap
		load.ClusterEndpoint = opts.ClusterEndpoint
	}

	return load.Validate()
}

func resolveChartSource(ctx context.Context, opts sharedOptions, deps UpgradeDeps) (resolutionResult, error) {
	hasChart := strings.TrimSpace(opts.ChartReference) != ""
	hasBundle := strings.TrimSpace(opts.BundlePath) != ""

	if err := validateResolveSource(hasChart, hasBundle); err != nil {
		return resolutionResult{}, err
	}
	if hasChart {
		return resolveFromChart(ctx, opts, deps)
	}
	return resolveFromBundle(ctx, opts, deps)
}

func validateResolveSource(hasChart, hasBundle bool) error {
	switch {
	case hasChart && hasBundle:
		return errConflictingSources
	case !hasChart && !hasBundle:
		return errMissingSource
	default:
		return nil
	}
}

func resolveFromChart(ctx context.Context, opts sharedOptions, deps UpgradeDeps) (resolutionResult, error) {
	if deps.Resolver == nil {
		return resolutionResult{}, errResolverPullerMissing
	}

	res, err := deps.Resolver.Resolve(ctx, helm.ResolveOptions{OCIReference: opts.ChartReference})
	if err != nil {
		return resolutionResult{}, err
	}

	return resolutionResult{Outcome: res}, nil
}

func resolveFromBundle(ctx context.Context, opts sharedOptions, deps UpgradeDeps) (resolutionResult, error) {
	if deps.Resolver != nil {
		res, err := deps.Resolver.Resolve(ctx, helm.ResolveOptions{
			BundlePath:     opts.BundlePath,
			BundleCacheDir: filepath.Dir(opts.BundlePath),
		})
		if err == nil && res.Bundle != nil {
			return resolutionResult{Outcome: res, Bundle: res.Bundle}, nil
		}
	}

	loader := deps.BundleLoader
	if loader == nil {
		loader = bundle.Load
	}

	bundleInst, err := loader(opts.BundlePath, filepath.Dir(opts.BundlePath))
	if err != nil {
		return resolutionResult{}, err
	}

	return resolutionResult{
		Outcome: helm.ResolveResult{
			Source: pkgstate.ChartSource{
				Type:      "bundle",
				Reference: opts.BundlePath,
			},
		},
		Bundle: bundleInst,
	}, nil
}

func resolveStateOverrides(opts sharedOptions) (pkgstate.Overrides, string, error) {
	resolver := internalstate.NewResolver()
	overrides := pkgstate.Overrides{
		StateFileName: opts.StateFileName,
		StateFilePath: opts.StateFilePath,
	}

	path, err := resolver.Resolve(overrides)
	if err != nil {
		return pkgstate.Overrides{}, "", err
	}

	return pkgstate.Overrides{StateFilePath: path}, path, nil
}

func deriveVersion(opts sharedOptions, res helm.ResolveResult) string {
	if opts.AppVersion != "" {
		return opts.AppVersion
	}
	if res.Source.Type == "oci" {
		if i := strings.LastIndex(res.Source.Reference, ":"); i != -1 && i+1 < len(res.Source.Reference) {
			return res.Source.Reference[i+1:]
		}
	}
	return ""
}

func workflowStepName(action appAction) string {
	switch action {
	case actionInstall:
		return stepAppInstall
	case actionUpgrade:
		return stepAppUpgrade
	default:
		return fmt.Sprintf("app-%s", action)
	}
}

func emitOutput(cmd *cobra.Command, profile *config.Profile, result helm.ResolveResult, statePath string, format string, action appAction, opts sharedOptions) error {
	title := ""
	switch action {
	case actionInstall:
		title = "Install"
	case actionUpgrade:
		title = "Upgrade"
	default:
		title = titleCaser.String(string(action))
	}

	switch format {
	case "text":
		fmt.Fprintf(cmd.OutOrStdout(), "%s completed successfully for release %s in namespace %s\n", title, profile.HelmRelease, profile.HelmNamespace)
		fmt.Fprintf(cmd.OutOrStdout(), "State written to %s\n", statePath)
		return nil
	case "json":
		payload := map[string]any{
			"status":    "success",
			"action":    string(action),
			"release":   profile.HelmRelease,
			"namespace": profile.HelmNamespace,
			"chart":     result.Source.Reference,
			"chartType": result.Source.Type,
			"stateFile": statePath,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		if profile.ClusterEndpoint != "" {
			payload["cluster"] = profile.ClusterEndpoint
		}
		if opts.AppVersion != "" {
			payload["version"] = opts.AppVersion
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetEscapeHTML(false)
		return enc.Encode(payload)
	default:
		return errUnsupportedOutput
	}
}

func buildResolveArgs(opts sharedOptions) []string {
	if strings.TrimSpace(opts.ChartReference) != "" {
		return []string{"helm", "pull", opts.ChartReference}
	}
	if strings.TrimSpace(opts.BundlePath) != "" {
		return []string{"bundle", "load", opts.BundlePath}
	}
	return []string{"helm", "pull"}
}

func buildResolveMetadata(res *helm.ResolveResult, opts sharedOptions) map[string]string {
	meta := map[string]string{}
	if res != nil {
		if res.Source.Type != "" {
			meta["source"] = res.Source.Type
		}
		if res.Source.Reference != "" {
			meta["reference"] = res.Source.Reference
		}
		if res.Source.Digest != "" {
			meta["digest"] = res.Source.Digest
		}
	} else {
		if strings.TrimSpace(opts.ChartReference) != "" {
			meta["reference"] = opts.ChartReference
		}
		if strings.TrimSpace(opts.BundlePath) != "" {
			meta["bundlePath"] = opts.BundlePath
		}
	}
	return meta
}

func buildHelmInstallMetadata(profile *config.Profile, res helm.ResolveResult) map[string]string {
	meta := map[string]string{
		"namespace": profile.HelmNamespace,
		"release":   profile.HelmRelease,
	}
	if res.Source.Type != "" {
		meta["source"] = res.Source.Type
	}
	if res.Source.Reference != "" {
		meta["reference"] = res.Source.Reference
	}
	if res.Source.Digest != "" {
		meta["digest"] = res.Source.Digest
	}
	return meta
}

func buildHelmInstallArgs(profile *config.Profile, opts sharedOptions) []string {
	args := []string{"helm", "upgrade", profile.HelmRelease}
	if ns := strings.TrimSpace(profile.HelmNamespace); ns != "" {
		args = append(args, "--namespace", ns)
	}
	if profile.EncryptedFile != "" {
		args = append(args, "--values", profile.EncryptedFile)
	}
	if profile.Passphrase != "" {
		args = append(args, "--values-passphrase", profile.Passphrase)
	}
	if strings.TrimSpace(opts.ChartReference) != "" {
		args = append(args, opts.ChartReference)
	}
	if strings.TrimSpace(opts.BundlePath) != "" {
		args = append(args, "--bundle-path", opts.BundlePath)
	}
	return args
}

func telemetryEmitterDefault(w io.Writer) (*telemetry.Emitter, error) {
	return telemetry.NewEmitter(w)
}
