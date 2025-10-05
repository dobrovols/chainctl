package cli

import (
	"github.com/spf13/cobra"

	appcmd "github.com/dobrovols/chainctl/cmd/chainctl/app"
	clustercmd "github.com/dobrovols/chainctl/cmd/chainctl/cluster"
	nodecmd "github.com/dobrovols/chainctl/cmd/chainctl/node"
	secretcmd "github.com/dobrovols/chainctl/cmd/chainctl/secrets"
)

// NewRootCommand constructs the root chainctl command.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chainctl",
		Short: "chainctl manages installation and lifecycle operations for the platform",
	}

	cmd.AddCommand(secretcmd.NewEncryptCommand())
	cmd.AddCommand(nodecmd.NewNodeCommand())
	cmd.AddCommand(clustercmd.NewClusterCommand())
	cmd.AddCommand(appcmd.NewAppCommand())

	return cmd
}
