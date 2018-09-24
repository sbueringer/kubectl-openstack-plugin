package cmd

import (
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/sbueringer/kubectl-openstack-plugin/pkg/output/mattermost"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd/api"

	"fmt"

	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/rest"
)

//TODO
type ServerOptions struct {
	configFlags *genericclioptions.ConfigFlags

	restConfig *rest.Config
	rawConfig  api.Config

	states string

	exporter string
	output   string
	args     []string

	genericclioptions.IOStreams
}

var (
	serverExample = `
	# list server
	%[1]s server
`
)

//TODO
func NewCmdServer(streams genericclioptions.IOStreams) *cobra.Command {
	o := &ServerOptions{
		configFlags: genericclioptions.NewConfigFlags(),
		IOStreams:   streams,
	}
	cmd := &cobra.Command{
		Use:          "server",
		Aliases:      []string{"srv"},
		Short:        "List all server from Kubernetes and OpenStack",
		Example:      fmt.Sprintf(serverExample, "kubectl os"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&o.states, "states", "", "filter by states, default list all")
	cmd.Flags().StringVarP(&o.exporter, "exporter", "e", "stdout", "stdout, mm or multiple (comma-separated)")
	cmd.Flags().StringVarP(&o.output, "output", "o", "markdown", "markdown or raw")
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

// Complete sets als necessary fields in VolumeOptions
func (o *ServerOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	var err error
	o.restConfig, err = o.configFlags.ToRawKubeConfigLoader().ClientConfig()
	if err != nil {
		return err
	}
	o.rawConfig, err = o.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}
	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *ServerOptions) Validate() error {
	if len(o.rawConfig.CurrentContext) == 0 {
		return errNoContext
	}

	return nil
}

// Run lists all server
func (o *ServerOptions) Run() error {
	if *o.configFlags.Context == "" {
		err := o.runWithConfig()
		if err != nil {
			return fmt.Errorf("error listing server for %s: %v\n", o.rawConfig.CurrentContext, err)
		}
		return nil
	}

	for context := range getMatchingContexts(o.rawConfig.Contexts, o.rawConfig.CurrentContext) {
		o.configFlags.Context = &context
		err := o.runWithConfig()
		if err != nil {
			fmt.Printf("Error listing server for %s: %v\n", context, err)
		}
	}
	return nil
}

func (o *ServerOptions) runWithConfig() error {
	kubeClient, err := getKubeClient(o.restConfig)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}
	osProvider, tenantID, err := getOpenStackClient(o.rawConfig)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	nodesMap, err := getNodes(kubeClient)
	if err != nil {
		return fmt.Errorf("error getting persistent volumes from Kubernetes: %v", err)
	}

	serversMap, err := getServer(osProvider)
	if err != nil {
		return fmt.Errorf("error getting servers from OpenStack: %v", err)
	}

	output, err := o.getPrettyServerList(nodesMap, serversMap)
	if err != nil {
		return fmt.Errorf("error creating output: %v", err)
	}

	for _, exporter := range strings.Split(o.exporter, ",") {
		switch exporter {
		case "stdout":
			{
				fmt.Printf(output)
			}
		case "mm":
			{
				var msg string
				switch o.output {
				case "raw":
					msg = fmt.Sprintf("Server for %s:\n\n````\n%s````\n\n", tenantID, output)
				case "markdown":
					msg = fmt.Sprintf("Server for %s:\n\n%s\n\n", tenantID, output)
				}
				mattermost.New().SendMessage(msg)
			}
		}
	}
	return nil
}

func (o *ServerOptions) getPrettyServerList(nodes map[string]v1.Node, server map[string]servers.Server) (string, error) {

	header := []string{"NODE_NAME", "STATUS", "KUBELET_VERSION", "KUBEPROXY_VERSION", "RUNTIME_VERSION", "DHC_VERSION", "SERVER_ID", "STATE", "CPU", "RAM", "IP"}

	var lines [][]string
	for _, s := range server {
		name := "-"
		status := "-"
		kubeletVersion := "-"
		kubeProxyVersion := "-"
		containerRuntimeVersion := "-"
		dhcVersion := "-"
		cpu := "-"
		ram := "-"
		ip := "-"
		if node, ok := nodes[s.ID]; ok {
			name = node.Name
			for _, st := range node.Status.Conditions {
				if st.Type == v1.NodeReady {
					status = "Ready"
					break
				}
			}
			if status == "" {
				status = "NotReady"
			}
			kubeletVersion = node.Status.NodeInfo.KubeletVersion
			kubeProxyVersion = node.Status.NodeInfo.KubeProxyVersion
			containerRuntimeVersion = node.Status.NodeInfo.ContainerRuntimeVersion
			dhcVersion = node.Labels["dhc-version"]
			cpu = node.Status.Capacity.Cpu().String()
			ram = fmt.Sprintf("%dMB", node.Status.Capacity.Memory().ScaledValue(resource.Mega))
			for _, addr := range node.Status.Addresses {
				if addr.Type == v1.NodeInternalIP {
					ip = addr.Address
				}
			}
		}

		matchesStates := false
		for _, state := range strings.Split(o.states, ",") {
			if s.Status == state {
				matchesStates = true
				break
			}
		}

		if matchesStates || o.states == "" {
			lines = append(lines, []string{name, status, kubeletVersion, kubeProxyVersion, containerRuntimeVersion, dhcVersion, s.ID, s.Status, cpu, ram, ip})
		}
	}
	return convertToTable(table{header, lines, 0, o.output})
}
