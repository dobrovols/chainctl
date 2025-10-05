package app

import "github.com/spf13/cobra"

// InstallOptions reuses upgrade options for install semantics.
type InstallOptions = UpgradeOptions

// InstallDeps reuses upgrade dependencies for install execution.
type InstallDeps = UpgradeDeps

// NewInstallCommand constructs the `chainctl app install` command.
func NewInstallCommand() *cobra.Command {
	opts := InstallOptions{}
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the micro-services application Helm release",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runAppAction(cmd, opts.shared(), defaultUpgradeDeps, actionInstall)
		},
	}

	bindCommonFlags(cmd, &opts)

	return cmd
}

// RunInstallForTest executes the install flow with injected dependencies.
func RunInstallForTest(cmd *cobra.Command, opts InstallOptions, deps InstallDeps) error {
	cmd.SilenceUsage = true
	return runAppAction(cmd, opts.shared(), deps, actionInstall)
}
