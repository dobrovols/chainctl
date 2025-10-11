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

	if strings.TrimSpace(options.ValuesFile) == "" {
		return errValuesFile
	}
	if action == actionUpgrade && strings.TrimSpace(options.ClusterEndpoint) == "" {
		return errClusterEndpoint
	}

	profile, err := buildProfileForAction(options, action)
	if err != nil {
		return err
	}

	stateOverrides, statePathHint, err := resolveStateOverrides(options)
	if err != nil {
		return err
	}

	emitter := deps.TelemetryEmitter
	if emitter == nil {
		emitter = telemetryEmitterDefault
	}
	tel, err := emitter(cmd.ErrOrStderr())
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

	workflowStep := workflowStepName(action)
	workflowMetadata := map[string]string{
		"mode":      string(profile.Mode),
		"namespace": profile.HelmNamespace,
		"release":   profile.HelmRelease,
	}
	if profile.ClusterEndpoint != "" {
		workflowMetadata["cluster"] = profile.ClusterEndpoint
	}
	logWorkflowStart(logger, workflowStep, workflowMetadata)
	defer func() {
		if err != nil {
			logWorkflowFailure(logger, workflowStep, workflowMetadata, err)
		}
	}()

	ctx := cmd.Context()
	resolveArgs := buildResolveArgs(options)
	resolved, err := resolveChartSource(ctx, options, deps)
	if err != nil {
		resolveMetadata := buildResolveMetadata(nil, options)
		logCommandEntry(logger, stepHelmResolve, resolveArgs, err.Error(), telemetry.SeverityError, resolveMetadata, err)
		return err
	}
	resolveMetadata := buildResolveMetadata(&resolved.Outcome, options)
	logCommandEntry(logger, stepHelmResolve, resolveArgs, "", telemetry.SeverityInfo, resolveMetadata, nil)
	if resolved.Outcome.Source.Type != "" {
		workflowMetadata["source"] = resolved.Outcome.Source.Type
	}
	if resolved.Outcome.Source.Digest != "" {
		workflowMetadata["digest"] = resolved.Outcome.Source.Digest
	}

	bundleInstance := resolved.Bundle

	helmHasLogging := false
	if installer, ok := deps.Installer.(*helm.Installer); ok {
		deps.Installer = installer.WithLogger(logger)
		helmHasLogging = true
	}

	helmMetadata := buildHelmInstallMetadata(profile, resolved.Outcome)
	helmArgs := buildHelmInstallArgs(profile, options)

	execute := func() error {
		installer := deps.Installer
		if installer == nil {
			installer = noopInstaller{}
		}
		installErr := installer.Install(profile, bundleInstance)
		if !helmHasLogging {
			severity := telemetry.SeverityInfo
			stderr := ""
			if installErr != nil {
				severity = telemetry.SeverityError
				stderr = installErr.Error()
			}
			logCommandEntry(logger, stepHelmCommand, helmArgs, stderr, severity, helmMetadata, installErr)
		}
		return installErr
	}

	if err := tel.EmitPhase(telemetry.PhaseHelm, helmMetadata, execute); err != nil {
		return err
	}

	record := pkgstate.Record{
		Release:         profile.HelmRelease,
		Namespace:       profile.HelmNamespace,
		Chart:           resolved.Outcome.Source,
		Version:         deriveVersion(options, resolved.Outcome),
		LastAction:      string(action),
		ClusterEndpoint: profile.ClusterEndpoint,
	}

	statePath, writeErr := deps.StateManager.Write(record, stateOverrides)
	if writeErr != nil {
		err = fmt.Errorf("state file could not be written: %w", writeErr)
		return err
	}

	if statePath == "" {
		statePath = statePathHint
	}

	logWorkflowSuccess(logger, workflowStep, workflowMetadata)
	return emitOutput(cmd, profile, resolved.Outcome, statePath, options.Output, action, options)
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

	switch {
	case hasChart && hasBundle:
		return resolutionResult{}, errConflictingSources
	case !hasChart && !hasBundle:
		return resolutionResult{}, errMissingSource
	case hasChart:
		if deps.Resolver == nil {
			return resolutionResult{}, errResolverPullerMissing
		}
		res, err := deps.Resolver.Resolve(ctx, helm.ResolveOptions{OCIReference: opts.ChartReference})
		if err != nil {
			return resolutionResult{}, err
		}
		return resolutionResult{Outcome: res}, nil
	default:
		if deps.Resolver != nil {
			res, err := deps.Resolver.Resolve(ctx, helm.ResolveOptions{BundlePath: opts.BundlePath, BundleCacheDir: filepath.Dir(opts.BundlePath)})
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
			Outcome: helm.ResolveResult{Source: pkgstate.ChartSource{Type: "bundle", Reference: opts.BundlePath}},
			Bundle:  bundleInst,
		}, nil
	}
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
