package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dobrovols/chainctl/pkg/tokens"
)

// JoinCommandOptions captures CLI flags.
type JoinCommandOptions struct {
	ClusterEndpoint string
	Role            string
	Token           string
	Labels          []string
	Taints          []string
	Output          string
}

// tokenConsumer defines the subset of store functionality needed for join flows.
type tokenConsumer interface {
	Consume(string, tokens.Scope) error
}

// NewJoinCommand returns the `chainctl node join` command.
func NewJoinCommand() *cobra.Command {
	opts := JoinCommandOptions{}

	cmd := &cobra.Command{
		Use:   "join",
		Short: "Join a node to the cluster using a pre-shared token",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runJoin(cmd, opts, joinStore())
		},
	}

	cmd.Flags().StringVar(&opts.ClusterEndpoint, "cluster-endpoint", "", "Kubernetes API endpoint")
	cmd.Flags().StringVar(&opts.Role, "role", "", "Node role: worker or control-plane")
	cmd.Flags().StringVar(&opts.Token, "token", "", "Pre-shared join token")
	cmd.Flags().StringSliceVar(&opts.Labels, "labels", nil, "Node labels key=value")
	cmd.Flags().StringSliceVar(&opts.Taints, "taints", nil, "Node taints key=value:effect")
	cmd.Flags().StringVar(&opts.Output, "output", "text", "Output format: text or json")

	return cmd
}

// RunJoinForTest executes join logic using provided store override.
func RunJoinForTest(cmd *cobra.Command, opts JoinCommandOptions, store tokenConsumer) error {
	return runJoin(cmd, opts, store)
}

func runJoin(cmd *cobra.Command, opts JoinCommandOptions, store tokenConsumer) error {
	if strings.TrimSpace(opts.ClusterEndpoint) == "" {
		return ErrClusterEndpoint()
	}
	scope, err := parseScope(opts.Role)
	if err != nil {
		return err
	}
	if strings.TrimSpace(opts.Token) == "" {
		return errTokenRequired
	}

	if err := store.Consume(opts.Token, scope); err != nil {
		return fmt.Errorf("validate token: %w", err)
	}

	switch opts.Output {
	case "json":
		payload := map[string]interface{}{
			"clusterEndpoint": opts.ClusterEndpoint,
			"role":            scope,
			"labels":          opts.Labels,
			"taints":          opts.Taints,
			"status":          "ready", // placeholder until actual join logic implemented
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	case "text":
		fmt.Fprintf(cmd.OutOrStdout(), "Validated token for role %s against cluster %s\n", scope, opts.ClusterEndpoint)
		return nil
	default:
		return fmt.Errorf("unsupported output format %q", opts.Output)
	}
}

var (
	errTokenRequired   = errors.New("token is required")
	errClusterEndpoint = errors.New("cluster endpoint is required")
)

// ErrTokenRequired exposes the sentinel.
func ErrTokenRequired() error { return errTokenRequired }

// ErrClusterEndpoint exposes the sentinel.
func ErrClusterEndpoint() error { return errClusterEndpoint }
