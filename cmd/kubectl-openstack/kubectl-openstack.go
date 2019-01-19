package main

import (
	"github.com/spf13/pflag"
	"os"

	"github.com/sbueringer/kubectl-openstack-plugin/pkg/cmd"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-openstack", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewCmdOpenStack(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
