package cluster

import "github.com/spf13/cobra"

// NewClusterCommand constructs the `chainctl cluster` parent command.
func NewClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manage cluster lifecycle operations",
	}

	cmd.AddCommand(NewInstallCommand())
	cmd.AddCommand(NewUpgradeCommand())
	return cmd
}
