package cmd

import (
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/sbueringer/kubectl-openstack-plugin/pkg/kubernetes"
	"github.com/sbueringer/kubectl-openstack-plugin/pkg/openstack"
	"github.com/sbueringer/kubectl-openstack-plugin/pkg/output"
	"github.com/sbueringer/kubectl-openstack-plugin/pkg/output/mattermost"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd/api"
	"os"
	"sort"

	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/rest"
)

//TODO
type ServerOptions struct {
	configFlags *genericclioptions.ConfigFlags

	rawConfig api.Config

	states string

	exporter   string
	output     string
	noHeader   bool
	args       []string
	onlyBroken bool
	debug      bool

	genericclioptions.IOStreams
}

var (
	serverExample = `
	# list server
	%[1]s server
	
	# list server with debug columns
	%[1]s server --debug
`
)

// NewCmdServer creates the server cmd
func NewCmdServer(streams genericclioptions.IOStreams) *cobra.Command {
	o := &ServerOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
	cmd := &cobra.Command{
		Use:          "server",
		Aliases:      []string{"srv"},
		Short:        "List all server from Kubernetes and OpenStack",
		Example:      fmt.Sprintf(serverExample, "kubectl openstack"),
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
	cmd.Flags().BoolVarP(&o.debug, "debug", "", false, "debug prints more columns")
	cmd.Flags().BoolVarP(&o.onlyBroken, "only-broken", "", false, "only show disks which are broken/out of sync")
	cmd.Flags().BoolVarP(&o.noHeader, "no-headers", "", false, "hide table headers")
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

// Run lists all server
func (o *ServerOptions) Run() error {
	contexts := kubernetes.GetMatchingContexts(o.rawConfig, *o.configFlags.Context)

	if len(contexts) == 1 {
		err := o.runWithConfig(contexts[0])
		if err != nil {
			return fmt.Errorf("error listing server for %s: %v\n", o.rawConfig.CurrentContext, err)
		}
		return nil
	}

	// multiple tenants
	// disable header here and print them once if required
	if !o.noHeader {
		var header []string
		if o.debug {
			header = serverDebugHeaders
		} else {
			header = serverHeaders
		}
		output, err := output.ConvertToTable(output.Table{header, [][]string{}, []int{0, 1}, o.output})
		if err != nil {
			return fmt.Errorf("error creating output: %v", err)
		}
		fmt.Printf(output)
	}
	o.noHeader = true
	for _, context := range contexts {
		o.configFlags.Context = &context
		err := o.runWithConfig(context)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing server for %s: %v\n", context, err)
		}
	}
	return nil
}

func (o *ServerOptions) runWithConfig(context string) error {
	if context == "" {
		return fmt.Errorf("no context set")
	}

	contextStruct := o.rawConfig.Contexts[context]
	cluster := o.rawConfig.Clusters[contextStruct.Cluster]
	authInfo := o.rawConfig.AuthInfos[contextStruct.AuthInfo]
	c := &rest.Config{
		Host: cluster.Server,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   cluster.CertificateAuthorityData,
			KeyData:  authInfo.ClientKeyData,
			CertData: authInfo.ClientCertificateData,
		},
	}

	kubeClient, err := kubernetes.GetKubeClient(c)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}
	osProvider, tenantID, err := openstack.GetOpenStackClient(context)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	nodesMap, err := kubernetes.GetNodes(kubeClient)
	if err != nil {
		return fmt.Errorf("error getting persistent volumes from Kubernetes: %v", err)
	}

	serversMap, err := openstack.GetServer(osProvider)
	if err != nil {
		return fmt.Errorf("error getting servers from OpenStack: %v", err)
	}

	output, err := o.getPrettyServerList(context, nodesMap, serversMap)
	if err != nil {
		return fmt.Errorf("error creating output: %v", err)
	}

	if output == "" {
		return nil
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

var serverHeaders = []string{"CLUSTER", "NODE_NAME", "STATUS", "KUBELET_VERSION", "KUBEPROXY_VERSION", "RUNTIME_VERSION", "DHC_VERSION", "SERVER_NAME", "SERVER_ID", "STATE", "CPU", "RAM", "IP", "NOTE"}
var serverDebugHeaders = []string{"CLUSTER", "NODE_NAME", "STATUS", "KUBELET_VERSION", "KUBEPROXY_VERSION", "RUNTIME_VERSION", "DHC_VERSION", "SERVER_NAME", "SERVER_ID", "VOLUMES", "STATE", "CPU", "RAM", "IP", "NOTE"}

func (o *ServerOptions) getPrettyServerList(context string, nodes map[string]v1.Node, server map[string]servers.Server) (string, error) {

	var header []string
	if !o.noHeader {
		if o.debug {
			header = serverDebugHeaders
		} else {
			header = serverHeaders
		}
	}

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
		attachmentCount := map[string]int{}
		var attachments []string
		var attachedVolumes []string
		overallNovaAttachmentCount := 0
		for _, attachedVolume := range s.AttachedVolumes {
			for key, volumeID := range attachedVolume {
				if key == "id" {
					attachmentCount[volumeID]++
					overallNovaAttachmentCount++
				}
			}
		}
		for a := range attachmentCount {
			attachments = append(attachments, a)
		}
		sort.Strings(attachments)
		for _, a := range attachments {
			attachedVolumes = append(attachedVolumes, fmt.Sprintf("%dx %s", attachmentCount[a], a))
		}
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

		var notes []string
		showDiskIfOnlyBroken := false
		// check error states
		if overallNovaAttachmentCount > len(attachedVolumes) {
			showDiskIfOnlyBroken = true
			notes = append(notes, "multiple attachments")
		}
		note := strings.Join(notes, ", ")

		if (!o.onlyBroken || showDiskIfOnlyBroken) && (matchesStates || o.states == "") {
			if o.debug {
				lines = append(lines, []string{context, name, status, kubeletVersion, kubeProxyVersion, containerRuntimeVersion, dhcVersion, s.Name, s.ID, strings.Join(attachedVolumes, " "), s.Status, cpu, ram, ip, note})
			} else {
				lines = append(lines, []string{context, name, status, kubeletVersion, kubeProxyVersion, containerRuntimeVersion, dhcVersion, s.Name, s.ID, s.Status, cpu, ram, ip, note})
			}
		}
	}
	if len(lines) > 0 {
		return output.ConvertToTable(output.Table{header, lines, []int{0, 1}, o.output})
	}
	return "", nil
}
