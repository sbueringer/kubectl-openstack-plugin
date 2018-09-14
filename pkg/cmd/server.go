package cmd

import (
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd/api"

	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

//TODO
type ServerOptions struct {
	configFlags *genericclioptions.ConfigFlags
	rawConfig   api.Config
	//TODO decide what todo with list
	list bool
	args []string

	genericclioptions.IOStreams
}

var (
	serverExample = `
	# list server
	%[1] server
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
	cmd.Flags().BoolVar(&o.list, "list", o.list, "if true, list")
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

// Complete sets als necessary fields in VolumeOptions
func (o *ServerOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	var err error
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

// Run lists all volumes
func (o *ServerOptions) Run() error {

	fmt.Printf("%t\n", o.list)

	kubeClient, err := getKubeClient(o.configFlags)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}
	osProvider, err := getOpenStackClient(o.rawConfig)
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

	output, err := getPrettyServerList(nodesMap, serversMap)
	if err != nil {
		return fmt.Errorf("error creating output: %v", err)
	}
	fmt.Printf(output)

	return nil
}

func getPrettyServerList(nodes map[string]v1.Node, server map[string]servers.Server) (string, error) {

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
			ram = fmt.Sprintf("%dG", node.Status.Capacity.Memory().ScaledValue(resource.Giga))
			for _, addr := range node.Status.Addresses {
				if addr.Type == v1.NodeInternalIP {
					ip = addr.Address
				}
			}
		}
		lines = append(lines, []string{name, status, kubeletVersion, kubeProxyVersion, containerRuntimeVersion, dhcVersion, s.ID, s.Status, cpu, ram, ip})
	}
	return printTable(table{header, lines, 0})
}
