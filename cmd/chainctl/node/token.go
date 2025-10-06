package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/dobrovols/chainctl/pkg/tokens"
)

// TokenCommandOptions bundles flag values for easier testing.
type TokenCommandOptions struct {
	Role        string
	TTL         string
	Description string
	Output      string
}

// tokenStore abstracts token persistence backends.
type tokenStore interface {
	Create(tokens.CreateOptions) (*tokens.CreatedToken, error)
}

// NewTokenCommand creates the `chainctl node token create` command.
func NewTokenCommand() *cobra.Command {
	opts := TokenCommandOptions{}

	cmd := &cobra.Command{
		Use:   "token",
		Short: "Manage node join tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runTokenCreate(cmd, opts, defaultStore())
		},
	}

	cmd.Flags().StringVar(&opts.Role, "role", "", "Node role: worker or control-plane")
	cmd.Flags().StringVar(&opts.TTL, "ttl", "2h", "Token time-to-live (e.g. 30m, 4h)")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Optional description for audit logging")
	cmd.Flags().StringVar(&opts.Output, "output", "text", "Output format: text or json")

	return cmd
}

// RunTokenCreateForTest executes the token creation workflow using the provided store.
func RunTokenCreateForTest(cmd *cobra.Command, opts TokenCommandOptions, store tokenStore) error {
	return runTokenCreate(cmd, opts, store)
}

func runTokenCreate(cmd *cobra.Command, opts TokenCommandOptions, store tokenStore) error {
	scope, err := parseScope(opts.Role)
	if err != nil {
		return err
	}

	ttl, err := time.ParseDuration(opts.TTL)
	if err != nil {
		return fmt.Errorf("invalid ttl: %w", err)
	}

	created, err := store.Create(tokens.CreateOptions{
		Scope:       scope,
		TTL:         ttl,
		CreatedBy:   os.Getenv("USER"),
		Description: opts.Description,
	})
	if err != nil {
		return err
	}

	switch opts.Output {
	case "json":
		payload := map[string]interface{}{
			"token":       created.Token,
			"tokenID":     created.ID,
			"scope":       created.Scope,
			"expiresAt":   created.ExpiresAt.Format(time.RFC3339),
			"description": created.Description,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	case "text":
		fmt.Fprintf(cmd.OutOrStdout(), "Token: %s\nExpiry: %s\nScope: %s\n", created.Token, created.ExpiresAt.Format(time.RFC3339), created.Scope)
		if created.Description != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Description: %s\n", created.Description)
		}
		return nil
	default:
		return fmt.Errorf("unsupported output format %q", opts.Output)
	}
}

var errInvalidRole = errors.New("role must be worker or control-plane")

// ErrInvalidRole exposes the sentinel error.
func ErrInvalidRole() error { return errInvalidRole }

func parseScope(raw string) (tokens.Scope, error) {
	switch raw {
	case "worker":
		return tokens.ScopeWorker, nil
	case "control-plane":
		return tokens.ScopeControlPlane, nil
	default:
		return "", errInvalidRole
	}
}
