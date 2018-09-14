package main

import (
	"os"

		"k8s.io/cli-runtime/pkg/genericclioptions"
	"github.com/sbueringer/kubectl-openstack-plugin/pkg/cmd"
)

func main() {
	//flags := pflag.NewFlagSet("kubectl-os", pflag.ExitOnError)
	//pflag.CommandLine = flags

	root := cmd.NewCmdOpenStack(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}