package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var errNoContext = fmt.Errorf("no context is currently set, use %q to select a new one", "kubectl config use-context <context>")

// NewCmdNamespace provides a cobra command
func NewCmdOpenStack(streams genericclioptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "os",
		Short:   "OpenStack commands for kubectl",
		Example: "",
		RunE: func(c *cobra.Command, args []string) error {
			return fmt.Errorf("subcommand is mandatory")
		},
	}
	genericclioptions.NewConfigFlags(true).AddFlags(cmd.Flags())

	cmd.AddCommand(NewCmdLB(streams))
	cmd.AddCommand(NewCmdServer(streams))
	cmd.AddCommand(NewCmdVolumes(streams))
	cmd.AddCommand(NewCmdVolumesFix(streams))
	cmd.AddCommand(NewCmdImportConfig(streams))
	return cmd
}
