package node

import "github.com/spf13/cobra"

// NewNodeCommand creates the `chainctl node` parent command.
func NewNodeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage cluster nodes",
	}

	cmd.AddCommand(NewTokenCommand())
	cmd.AddCommand(NewJoinCommand())

	return cmd
}
