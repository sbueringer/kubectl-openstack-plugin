package cmd

import (
	"fmt"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/sbueringer/kubectl-openstack-plugin/pkg/kubernetes"
	"github.com/sbueringer/kubectl-openstack-plugin/pkg/openstack"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd/api"
)

//TODO
type VolumesFixOptions struct {
	configFlags *genericclioptions.ConfigFlags

	rawConfig api.Config

	args                   []string
	detachCinder           bool
	detachNova             bool
	force                  bool
	attachNova             string
	attachCinder           string
	attachCinderMountpoint string

	genericclioptions.IOStreams
}

var (
	volumesFixExample = `
	# detach disk in Cinder
	%[1]s volumes-fix <volumes-id> --detach-cinder
	
	# detach disk in Nova
	%[1]s volumes-fix <volumes-id> --detach-nova
`
)

//TODO
func NewCmdVolumesFix(streams genericclioptions.IOStreams) *cobra.Command {
	o := &VolumesFixOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   streams,
	}
	cmd := &cobra.Command{
		Use:          "volumes-fix",
		Aliases:      []string{"vsf"},
		Short:        "Fix volumes from Kubernetes and OpenStack",
		Example:      fmt.Sprintf(volumesFixExample, "kubectl openstack"),
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
	// See also
	// Nova: https://developer.openstack.org/api-ref/compute/?expanded=detach-a-volume-from-an-instance-detail#detach-a-volume-from-an-instance
	// Cinder: https://developer.openstack.org/api-ref/block-storage/v3/index.html?expanded=detach-volume-from-server-detail#volume-actions-volumes-action
	// https://raymii.org/s/articles/Fix_inconsistent_Openstack_volumes_and_instances_from_Cinder_and_Nova_via_the_database.html
	cmd.Flags().BoolVarP(&o.detachCinder, "detach-cinder", "", false, "Detach the disk in Cinder. Be careful this does not remove the attachment from the server in Nova.")
	cmd.Flags().StringVarP(&o.attachCinder, "attach-cinder", "", "", "")
	cmd.Flags().StringVarP(&o.attachCinderMountpoint, "attach-cinder-mountpoint", "", "", "")

	cmd.Flags().BoolVarP(&o.detachNova, "detach-nova", "", false, "Detach disk in Nova. This only works if the volume is really attached (so it doesn't when cinder shows no attachments to this server).")
	cmd.Flags().StringVarP(&o.attachNova, "attach-nova", "", "", "")
	cmd.Flags().BoolVarP(&o.force, "force", "f", false, "Currently only affects detach-cinder. Use force-detach.")
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

// Complete sets als necessary fields in VolumeOptions
func (o *VolumesFixOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	var err error
	o.rawConfig, err = o.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}
	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *VolumesFixOptions) Validate() error {
	if len(o.rawConfig.CurrentContext) == 0 {
		return errNoContext
	}

	return nil
}

// Run lists all volumes
func (o *VolumesFixOptions) Run() error {
	contexts := kubernetes.GetMatchingContexts(o.rawConfig, *o.configFlags.Context)

	if len(contexts) == 1 {
		err := o.runWithConfig(contexts[0])
		if err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("found multiple contexts: %v", contexts)
}

func (o *VolumesFixOptions) runWithConfig(context string) error {
	if context == "" {
		return fmt.Errorf("no context set")
	}

	osProvider, _, err := openstack.GetOpenStackClient(context)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	volumesMap, err := openstack.GetVolumes(osProvider)
	if err != nil {
		return fmt.Errorf("error getting volumes from OpenStack: %v", err)
	}

	serversMap, err := openstack.GetServer(osProvider)
	if err != nil {
		return fmt.Errorf("error getting servers from OpenStack: %v", err)
	}

	// loop over volumes
	for _, vID := range o.args {
		volume, ok := volumesMap[vID]
		if !ok {
			fmt.Printf("Volume with id %s not found\n", vID)
		}

		var srvs []servers.Server
		for _, srv := range serversMap {
			for _, attachedVolume := range srv.AttachedVolumes {
				if attachedVolume.ID == vID {
					srvs = append(srvs, srv)
				}
			}
		}

		if o.attachCinder != "" && o.attachCinderMountpoint != "" {
			err := openstack.AttachVolumeCinder(osProvider, volume.ID, o.attachCinder, o.attachCinderMountpoint)
			if err != nil {
				return err
			}
		}
		if o.attachNova != "" {
			err := openstack.AttachVolumeNova(osProvider, volume.ID, o.attachNova)
			if err != nil {
				return err
			}
		}
		if o.detachCinder {
			err := openstack.DetachVolumeCinder(osProvider, volume.ID, o.force)
			if err != nil {
				return err
			}
		}
		if o.detachNova {
			uniqueServerIDs := map[string]bool{}
			for _, srv := range srvs {
				uniqueServerIDs[srv.ID] = true
			}
			for srvID := range uniqueServerIDs {
				err := openstack.DetachVolumeNova(osProvider, volume.ID, srvID)
				if err != nil {
					return err
				}
			}
		}
	}

	//fmt.Printf("%v\n", tenantID)
	//fmt.Printf("%v\n", volumesMap)
	//fmt.Printf("%v\n", serversMap)

	return nil
}
