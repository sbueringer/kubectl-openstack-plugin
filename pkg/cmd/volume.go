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
)

//TODO
type VolumesOptions struct {
	configFlags *genericclioptions.ConfigFlags
	rawConfig   api.Config
	//TODO decide what todo with list
	list bool
	args []string

	genericclioptions.IOStreams
}

var (
	volumesExample = `
	# list volumes
	%[1] volumes
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
	cmd.Flags().BoolVar(&o.list, "list", o.list, "if true, list")
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

// Complete sets als necessary fields in VolumeOptions
func (o *VolumesOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	var err error
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

	fmt.Printf("%t\n", o.list)

	kubeClient, err := getKubeClient(o.configFlags)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}
	osProvider, err := getOpenStackClient(o.rawConfig)
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

	output, err := getPrettyVolumeList(pvMap, volumesMap, serversMap)
	if err != nil {
		return fmt.Errorf("error creating ouput: %v", err)
	}
	fmt.Printf(output)

	return nil
}

func getPrettyVolumeList(pvs map[string]v1.PersistentVolume, volumes map[string]volumes.Volume, server map[string]servers.Server) (string, error) {

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
		lines = append(lines, []string{pvClaim, pvName, v.ID, strings.Join(attachServers, " "), v.Status})
	}
	return printTable(table{header, lines, 0})
}
