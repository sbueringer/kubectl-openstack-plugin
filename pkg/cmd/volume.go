package cmd

import (
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/sbueringer/kubectl-openstack-plugin/pkg/kubernetes"
	"github.com/sbueringer/kubectl-openstack-plugin/pkg/openstack"
	"github.com/sbueringer/kubectl-openstack-plugin/pkg/output"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd/api"
	"os"

	"fmt"
	"strings"

	"github.com/sbueringer/kubectl-openstack-plugin/pkg/output/mattermost"
	"k8s.io/client-go/rest"
)

//TODO
type VolumesOptions struct {
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
	volumesExample = `
	# list volumes
	%[1]s volumes
	
	# list volumes with debug columns
	%[1]s volumes --debug
`
)

// NewCmdVolumes creates the volumes cmd
func NewCmdVolumes(streams genericclioptions.IOStreams) *cobra.Command {
	o := &VolumesOptions{
		configFlags: genericclioptions.NewConfigFlags(),
		IOStreams:   streams,
	}
	cmd := &cobra.Command{
		Use:          "volumes",
		Aliases:      []string{"vs"},
		Short:        "List all volumes from Kubernetes and OpenStack",
		Example:      fmt.Sprintf(volumesExample, "kubectl openstack"),
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
	contexts := kubernetes.GetMatchingContexts(o.rawConfig, *o.configFlags.Context)

	if len(contexts) == 1 {
		err := o.runWithConfig(contexts[0])
		if err != nil {
			return fmt.Errorf("error listing volumes for %s: %v\n", o.rawConfig.CurrentContext, err)
		}
		return nil
	}

	// multiple tenants
	// disable header here and print them once if required
	if !o.noHeader {
		var header []string
		if o.debug {
			header = volumeDebugHeaders
		} else {
			header = volumeHeaders
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
			fmt.Fprintf(os.Stderr, "Error listing volumes for %s: %v\n", context, err)
		}
	}
	return nil
}

func (o *VolumesOptions) runWithConfig(context string) error {
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

	pvMap, err := kubernetes.GetPersistentVolumes(kubeClient)
	if err != nil {
		return fmt.Errorf("error getting persistent volumes from Kubernetes: %v", err)
	}

	podMap, err := kubernetes.GetPodsByPVC(kubeClient)
	if err != nil {
		return fmt.Errorf("error getting persistent volumes from Kubernetes: %v", err)
	}

	volumesMap, err := openstack.GetVolumes(osProvider)
	if err != nil {
		return fmt.Errorf("error getting volumes from OpenStack: %v", err)
	}

	serversMap, err := openstack.GetServer(osProvider)
	if err != nil {
		return fmt.Errorf("error getting servers from OpenStack: %v", err)
	}

	output, err := o.getPrettyVolumeList(context, pvMap, podMap, volumesMap, serversMap)
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
					msg = fmt.Sprintf("Volumes for %s:\n\n````\n%s````\n\n", tenantID, output)
				case "markdown":
					msg = fmt.Sprintf("Volumes for %s:\n\n%s\n\n", tenantID, output)
				}
				mattermost.New().SendMessage(msg)
			}
		}
	}
	return nil
}

var volumeHeaders = []string{"CLUSTER", "PVC", "POD", "POD_NODE", "POD_STATUS", "CINDER_NAME", "CINDER_ID", "CINDER_SERVER", "CINDER_SERVER_ID", "CINDER_STATUS"}
var volumeDebugHeaders = []string{"CLUSTER", "PVC", "PV", "POD", "POD_NODE", "POD_STATUS", "CINDER_NAME", "CINDER_ID", "CINDER_SERVER", "CINDER_SERVER_ID", "CINDER_STATUS", "NOVA_SERVER", "NOVA_SERVER_ID", "NOTE"}

func (o *VolumesOptions) getPrettyVolumeList(context string, pvs map[string]v1.PersistentVolume, podMap map[string][]v1.Pod, volumes map[string]volumes.Volume, server map[string]servers.Server) (string, error) {

	var header []string
	if !o.noHeader {
		if o.debug {
			header = volumeDebugHeaders
		} else {
			header = volumeHeaders
		}
	}

	var lines [][]string
	for _, v := range volumes {
		var cinderServers []string
		var cinderServerIDs []string
		var novaServers []string
		var novaServerIDs []string
		for _, a := range v.Attachments {
			cinderServerIDs = append(cinderServerIDs, a.ServerID)
			if srv, ok := server[a.ServerID]; ok {
				cinderServers = append(cinderServers, srv.Name)
			} else {
				cinderServers = append(cinderServers, "not found")
			}
		}
		overallNovaAttachmentCount := 0
		for _, srv := range server {
			count := 0
			for _, attachedVolume := range srv.AttachedVolumes {
				for key, volumeID := range attachedVolume {
					if key == "id" && volumeID == v.ID {
						count++
					}
				}
			}
			if count > 0 {
				novaServers = append(novaServers, fmt.Sprintf(" %dx %s", count, srv.Name))
				novaServerIDs = append(novaServerIDs, fmt.Sprintf(" %dx %s", count, srv.ID))
				overallNovaAttachmentCount += count
			}
		}
		pvName := "-"
		pvClaim := "-"
		podName := "-"
		podNode := "-"
		podStatus := "-"
		if pv, ok := pvs[v.ID]; ok {
			pvName = pv.Name
			pvClaim = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
			if pods, ok := podMap[pvClaim]; ok {
				pod := kubernetes.FindNotEvictedPod(pods)
				podName = pod.Name
				podStatus = kubernetes.GetPodStatus(pod)
				podNode = pod.Spec.NodeName
			}
		}

		matchesStates := false
		for _, state := range strings.Split(o.states, ",") {
			if v.Status == state {
				matchesStates = true
				break
			}
		}

		var notes []string
		showDiskIfOnlyBroken := false
		// check error states
		if overallNovaAttachmentCount >= 2 {
			showDiskIfOnlyBroken = true
			notes = append(notes, "multiple attachments")
		}
		if podNode != "-" && podStatus != "Completed" && !strings.Contains(strings.Join(cinderServers, " "), podNode) {
			showDiskIfOnlyBroken = true
			notes = append(notes, "pod != cinder server")
		}
		if podNode != "-" && podStatus != "Completed" && !strings.Contains(strings.Join(novaServers, " "), podNode) {
			showDiskIfOnlyBroken = true
			notes = append(notes, "pod != nova server")
		}
		if !strings.Contains(strings.Join(novaServers, " "), strings.Join(cinderServers, " ")) {
			showDiskIfOnlyBroken = true
			notes = append(notes, "nova != cinder server")
		}
		if v.Status == "available" && (len(novaServers) > 0 || len(cinderServers) > 0) {
			showDiskIfOnlyBroken = true
			notes = append(notes, "available but attached")
		}
		if v.Status == "available" && podName != "-" && podStatus != "Completed" {
			showDiskIfOnlyBroken = true
			notes = append(notes, fmt.Sprintf("available but pod %q", podStatus))
		}
		if v.Status == "in-use" && (len(novaServers) == 0 || len(cinderServers) == 0) {
			showDiskIfOnlyBroken = true
			notes = append(notes, "in-use but not attached")
		}
		if strings.Contains(strings.Join(cinderServers, " "), "not found") {
			showDiskIfOnlyBroken = true
			notes = append(notes, "attached server not found")
		}
		if pvClaim == "-" && pvName == "-" && podName == "-" && strings.HasPrefix(v.Name, "kubernetes-dynamic-pvc")  {
			showDiskIfOnlyBroken = true
			notes = append(notes, "kubernetes disk has no pv/pvc/pod")
		}
		note := strings.Join(notes, ", ")

		if (!o.onlyBroken || showDiskIfOnlyBroken) && (matchesStates || o.states == "") {
			if o.debug {
				lines = append(lines, []string{context, pvClaim, pvName, podName, podNode, podStatus,
					v.Name, v.ID, strings.Join(cinderServers, " "), strings.Join(cinderServerIDs, " "), v.Status,
					strings.Join(novaServers, " "), strings.Join(novaServerIDs, " "), note,
				})
			} else {
				lines = append(lines, []string{context, pvClaim, podName, podNode, podStatus,
					v.Name, v.ID, strings.Join(cinderServers, " "), strings.Join(cinderServerIDs, " "), v.Status,
				})
			}
		}
	}
	if len(lines) > 0 {
		return output.ConvertToTable(output.Table{header, lines, []int{0, 1}, o.output})
	}
	return "", nil
}
