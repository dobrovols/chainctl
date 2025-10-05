package app

import "github.com/spf13/cobra"

// NewAppCommand constructs the `chainctl app` parent command.
func NewAppCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Manage application lifecycle operations",
	}

	cmd.AddCommand(NewInstallCommand())
	cmd.AddCommand(NewUpgradeCommand())

	return cmd
}
