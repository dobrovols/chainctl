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

func runAppAction(cmd *cobra.Command, options sharedOptions, deps UpgradeDeps, action appAction) error {
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

	ctx := cmd.Context()
	resolved, err := resolveChartSource(ctx, options, deps)
	if err != nil {
		return err
	}

	bundleInstance := resolved.Bundle

	metadata := map[string]string{
		"mode":   string(profile.Mode),
		"source": resolved.Outcome.Source.Type,
	}
	if ns := profile.HelmNamespace; ns != "" {
		metadata["namespace"] = ns
	}
	if digest := resolved.Outcome.Source.Digest; digest != "" {
		metadata["digest"] = digest
	}

	emitter := deps.TelemetryEmitter
	if emitter == nil {
		emitter = telemetryEmitterDefault
	}
	tel := emitter(cmd.ErrOrStderr())

	execute := func() error {
		installer := deps.Installer
		if installer == nil {
			installer = noopInstaller{}
		}
		return installer.Install(profile, bundleInstance)
	}

	if err := tel.EmitPhase(telemetry.PhaseHelm, metadata, execute); err != nil {
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

	statePath, err := deps.StateManager.Write(record, stateOverrides)
	if err != nil {
		return fmt.Errorf("state file could not be written: %w", err)
	}

	if statePath == "" {
		statePath = statePathHint
	}

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

func telemetryEmitterDefault(w io.Writer) *telemetry.Emitter {
	return telemetry.NewEmitter(w)
}
