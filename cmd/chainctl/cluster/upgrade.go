package cluster

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dobrovols/chainctl/internal/config"
	"github.com/dobrovols/chainctl/pkg/telemetry"
	"github.com/dobrovols/chainctl/pkg/upgrade"
)

// UpgradeOptions captures cluster upgrade flags.
type UpgradeOptions struct {
	ClusterEndpoint    string
	K3sVersion         string
	ControllerManifest string
	AirgappedBundle    string
	Output             string
}

// UpgradePlanner orchestrates system-upgrade-controller operations.
type UpgradePlanner interface {
	PlanUpgrade(*config.Profile, upgrade.Plan) error
}

// UpgradeDeps bundles dependencies for the upgrade command.
type UpgradeDeps struct {
	Planner          UpgradePlanner
	TelemetryEmitter func(io.Writer) *telemetry.Emitter
}

var (
	errClusterEndpointRequired = errors.New("cluster endpoint is required")
	errK3sVersionRequired      = errors.New("k3s version is required")
)

// ErrClusterEndpoint exposes the sentinel.
func ErrClusterEndpoint() error { return errClusterEndpointRequired }

// ErrK3sVersion exposes the sentinel.
func ErrK3sVersion() error { return errK3sVersionRequired }

// defaultUpgradeDeps for production.
var defaultUpgradeDeps = UpgradeDeps{
	Planner:          upgrade.NewPlanner(nil),
	TelemetryEmitter: telemetry.NewEmitter,
}

// NewUpgradeCommand constructs `chainctl cluster upgrade`.
func NewUpgradeCommand() *cobra.Command {
	opts := UpgradeOptions{}
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade k3s using system-upgrade-controller",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runClusterUpgrade(cmd, opts, defaultUpgradeDeps)
		},
	}

	cmd.Flags().StringVar(&opts.ClusterEndpoint, "cluster-endpoint", "", "Cluster API endpoint")
	cmd.Flags().StringVar(&opts.K3sVersion, "k3s-version", "", "Target k3s version")
	cmd.Flags().StringVar(&opts.ControllerManifest, "controller-manifest", "", "Path or URL to controller manifest override")
	cmd.Flags().StringVar(&opts.AirgappedBundle, "bundle-path", "", "Air-gapped bundle to mount for controller assets")
	cmd.Flags().StringVar(&opts.Output, "output", "text", "Output format: text or json")

	return cmd
}

// RunClusterUpgradeForTest executes the upgrade workflow with injected dependencies.
func RunClusterUpgradeForTest(cmd *cobra.Command, opts UpgradeOptions, deps UpgradeDeps) error {
	return runClusterUpgrade(cmd, opts, deps)
}

func runClusterUpgrade(cmd *cobra.Command, opts UpgradeOptions, deps UpgradeDeps) error {
	if strings.TrimSpace(opts.ClusterEndpoint) == "" {
		return errClusterEndpointRequired
	}
	if strings.TrimSpace(opts.K3sVersion) == "" {
		return errK3sVersionRequired
	}

	profile := &config.Profile{
		Mode:            config.ModeReuse,
		ClusterEndpoint: opts.ClusterEndpoint,
	}

	plan := upgrade.Plan{
		K3sVersion:         opts.K3sVersion,
		ControllerManifest: opts.ControllerManifest,
		AirgappedBundle:    opts.AirgappedBundle,
	}

	emitter := deps.TelemetryEmitter
	if emitter == nil {
		emitter = telemetry.NewEmitter
	}
	planner := deps.Planner
	if planner == nil {
		planner = upgrade.NewPlanner(nil)
	}

	tel := emitter(cmd.OutOrStdout())

	if err := tel.EmitPhase(telemetry.PhaseUpgrade, map[string]string{"version": opts.K3sVersion}, func() error {
		return planner.PlanUpgrade(profile, plan)
	}); err != nil {
		return err
	}

	return emitUpgradeOutput(cmd, profile, plan, opts.Output)
}

func emitUpgradeOutput(cmd *cobra.Command, profile *config.Profile, plan upgrade.Plan, format string) error {
	switch format {
	case "text":
		fmt.Fprintf(cmd.OutOrStdout(), "Cluster upgrade scheduled for %s to version %s\n", profile.ClusterEndpoint, plan.K3sVersion)
		return nil
	case "json":
		payload := map[string]interface{}{
			"status":     "scheduled",
			"cluster":    profile.ClusterEndpoint,
			"k3sVersion": plan.K3sVersion,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		}
		if plan.ControllerManifest != "" {
			payload["controllerManifest"] = plan.ControllerManifest
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(payload)
	default:
		return errUnsupportedOutput
	}
}
