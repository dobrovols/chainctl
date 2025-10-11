package cluster

import (
	"github.com/spf13/cobra"

	"github.com/dobrovols/chainctl/cmd/chainctl/declarative"
)

func markDeclarative(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[declarative.AnnotationEnabled] = "true"
}
