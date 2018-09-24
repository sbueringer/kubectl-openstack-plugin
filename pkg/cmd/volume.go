package cmd

import (
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd/api"

	"fmt"
	"strings"

	"github.com/sbueringer/kubectl-openstack-plugin/pkg/output/mattermost"
	"k8s.io/client-go/rest"
)

//TODO
type VolumesOptions struct {
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
	volumesExample = `
	# list volumes
	%[1]s volumes
`
)

//TODO
func NewCmdVolumes(streams genericclioptions.IOStreams) *cobra.Command {
	o := &VolumesOptions{
		configFlags: genericclioptions.NewConfigFlags(),
		IOStreams:   streams,
	}
	cmd := &cobra.Command{
		Use:          "volumes",
		Aliases:      []string{"vs"},
		Short:        "List all volumes from Kubernetes and OpenStack",
		Example:      fmt.Sprintf(volumesExample, "kubectl os"),
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
func (o *VolumesOptions) Complete(cmd *cobra.Command, args []string) error {
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
func (o *VolumesOptions) Validate() error {
	if len(o.rawConfig.CurrentContext) == 0 {
		return errNoContext
	}

	return nil
}

// Run lists all volumes
func (o *VolumesOptions) Run() error {
	if *o.configFlags.Context == "" {
		err := o.runWithConfig()
		if err != nil {
			return fmt.Errorf("error listing volumes for %s: %v\n", o.rawConfig.CurrentContext, err)
		}
		return nil
	}

	for context := range getMatchingContexts(o.rawConfig.Contexts, *o.configFlags.Context) {
		o.configFlags.Context = &context
		err := o.runWithConfig()
		if err != nil {
			fmt.Printf("Error listing volumes for %s: %v\n", context, err)
		}
	}
	return nil
}

func (o *VolumesOptions) runWithConfig() error {
	kubeClient, err := getKubeClient(o.restConfig)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}
	osProvider, tenantID, err := getOpenStackClient(o.rawConfig)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	pvMap, err := getPersistentVolumes(kubeClient)
	if err != nil {
		return fmt.Errorf("error getting persistent volumes from Kubernetes: %v", err)
	}

	volumesMap, err := getVolumes(osProvider)
	if err != nil {
		return fmt.Errorf("error getting volumes from OpenStack: %v", err)
	}

	serversMap, err := getServer(osProvider)
	if err != nil {
		return fmt.Errorf("error getting servers from OpenStack: %v", err)
	}

	output, err := o.getPrettyVolumeList(pvMap, volumesMap, serversMap)
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
					msg = fmt.Sprintf("Volumes for %s:\n\n````\n%s````\n", tenantID, output)
				case "markdown":
					msg = fmt.Sprintf("Volumes for %s:\n\n%s\n", tenantID, output)
				}
				mattermost.New().SendMessage(msg)
			}
		}
	}
	return nil
}

func (o *VolumesOptions) getPrettyVolumeList(pvs map[string]v1.PersistentVolume, volumes map[string]volumes.Volume, server map[string]servers.Server) (string, error) {

	header := []string{"CLAIM", "PV_NAME", "CINDER_ID", "SERVERS", "STATUS"}

	var lines [][]string
	for _, v := range volumes {
		var attachServers []string
		for _, a := range v.Attachments {
			if srv, ok := server[a.ServerID]; ok {
				attachServers = append(attachServers, srv.Name)
			}
		}
		pvName := "-"
		pvClaim := "-"
		if pv, ok := pvs[v.ID]; ok {
			pvName = pv.Name
			pvClaim = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)

		}

		matchesStates := false
		for _, state := range strings.Split(o.states, ",") {
			if v.Status == state {
				matchesStates = true
				break
			}
		}

		if matchesStates || o.states == "" {
			lines = append(lines, []string{pvClaim, pvName, v.ID, strings.Join(attachServers, " "), v.Status})
		}
	}
	return convertToTable(table{header, lines, 0, o.output})
}
