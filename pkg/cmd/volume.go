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
	namespaces string

	exporter   string
	output     string
	noHeader   bool
	columns    string
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
		configFlags: genericclioptions.NewConfigFlags(true),
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
	cmd.Flags().StringVar(&o.namespaces, "namespaces", "n", "filter by Kubernetes namespaces, default list all")
	cmd.Flags().StringVarP(&o.exporter, "exporter", "e", "stdout", "stdout, mm or multiple (comma-separated)")
	cmd.Flags().StringVarP(&o.output, "output", "o", "markdown", "markdown or raw")
	cmd.Flags().BoolVarP(&o.debug, "debug", "", false, "debug prints debug columns, equivalent to --columns=DEBUG")
	cmd.Flags().BoolVarP(&o.onlyBroken, "only-broken", "", false, "only show disks which are broken/out of sync")
	cmd.Flags().BoolVarP(&o.noHeader, "no-headers", "", false, "hide table headers")
	cmd.Flags().StringVar(&o.columns, "columns", strings.Join(defaultHeaders, ","), fmt.Sprintf("column-separated list of headers to show, if set to DEBUG a special debug subset of columns is shown (%q). The following columns are available: %q", strings.Join(debugHeaders, ","), strings.Join(allHeaders, ",")))
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

var defaultHeaders = []string{"PVC", "POD", "POD_NODE", "POD_STATUS", "CINDER_NAME", "SIZE", "CINDER_ID", "CINDER_SERVER", "CINDER_SERVER_ID", "CINDER_STATUS"}
var debugHeaders = []string{"PVC", "PV", "POD", "POD_NODE", "POD_STATUS", "CINDER_NAME", "CINDER_ID", "CINDER_SERVER", "CINDER_STATUS", "NOVA_SERVER", "NOTE"}
var allHeaders = []string{"CLUSTER", "PVC", "PV", "POD", "POD_NODE", "POD_STATUS", "CINDER_NAME", "SIZE", "CINDER_ID", "CINDER_SERVER", "CINDER_SERVER_ID", "CINDER_STATUS", "NOVA_SERVER", "NOVA_SERVER_ID", "NOTE"}

// Complete sets als necessary fields in VolumeOptions
func (o *VolumesOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	var err error
	o.rawConfig, err = o.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}
	if o.debug || o.columns == "DEBUG" {
		o.columns = strings.Join(debugHeaders, ",")
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
		output, err := output.ConvertToTable(output.Table{strings.Split(o.columns, ","), [][]string{}, []int{0, 1}, o.output})
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

	attachmentsMap, err := openstack.GetVolumeAttachmentsForServerNova(osProvider, serversMap)
	if err != nil {
		return fmt.Errorf("error getting attachments from OpenStack: %v", err)
	}

	output, err := o.getPrettyVolumeList(context, pvMap, podMap, volumesMap, serversMap, attachmentsMap)
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

func (o *VolumesOptions) getPrettyVolumeList(context string, pvs map[string]v1.PersistentVolume, podMap map[string][]v1.Pod, volumes map[string]volumes.Volume, server map[string]servers.Server, attachmentsMap map[string]*openstack.NovaVolumeAttachments) (string, error) {

	var header []string
	if !o.noHeader {
		header = strings.Split(o.columns, ",")
	}

	linesAllColumns := []map[string]string{}
	for _, v := range volumes {

		// Skip disk if it's status doesn't match one of the states defined in the state flag
		if o.states != "" {
			var matchesStates bool
			for _, state := range strings.Split(o.states, ",") {
				if v.Status == state {
					matchesStates = true
					break
				}
			}
			if !matchesStates {
				continue
			}
		}

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
			var devices []string
			for _, attachedVolume := range attachmentsMap[srv.ID].VolumeAttachments {
				if attachedVolume.VolumeID == v.ID {
					count++
					devices = append(devices, attachedVolume.Device)
				}
			}
			if count > 0 {
				novaServers = append(novaServers, fmt.Sprintf(" %dx %s:%v", count, srv.Name, devices))
				novaServerIDs = append(novaServerIDs, fmt.Sprintf(" %dx %s", count, srv.ID))
				overallNovaAttachmentCount += count
			}
		}
		pvName := "-"
		pvClaim := "-"
		var pods []v1.Pod
		if pv, ok := pvs[v.ID]; ok {
			pvName = pv.Name
			pvClaim = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
			if allPods, ok := podMap[pvClaim]; ok {
				pods = kubernetes.FindNotEvictedPods(allPods)
			}
			if o.namespaces != "" {
				var matchesNamespaces bool
				for _, namespace := range strings.Split(o.namespaces, ",") {
					if pv.Spec.ClaimRef.Namespace == namespace {
						matchesNamespaces = true
						break
					}
				}
				if !matchesNamespaces {
					continue
				}
			}
		}

		if len(pods) == 0 {
			linesAllColumns = append(linesAllColumns, createLine(v, context, pvClaim, pvName, nil, overallNovaAttachmentCount, cinderServers, cinderServerIDs, novaServers, novaServerIDs))
		} else {
			for _, pod := range pods {
				linesAllColumns = append(linesAllColumns, createLine(v, context, pvClaim, pvName, &pod, overallNovaAttachmentCount, cinderServers, cinderServerIDs, novaServers, novaServerIDs))
			}
		}
	}

	var lines [][]string
	for _, allColumns := range linesAllColumns {
		_, containsNote := allColumns["NOTE"]
		if !o.onlyBroken || containsNote {
			var lineColumns []string
			for _, column := range strings.Split(o.columns, ",") {
				lineColumns = append(lineColumns, allColumns[column])
			}
			lines = append(lines, lineColumns)
		}
	}
	if len(lines) > 0 {
		return output.ConvertToTable(output.Table{header, lines, []int{0, 1, 2}, o.output})
	}
	return "", nil
}

func createLine(v volumes.Volume, context, pvClaim string, pvName string, pod *v1.Pod, overallNovaAttachmentCount int, cinderServers []string, cinderServerIDs []string, novaServers []string, novaServerIDs []string) map[string]string {

	podName := "-"
	podStatus := "-"
	podNode := "-"
	if pod != nil {
		podName = pod.Name
		podStatus = kubernetes.GetPodStatus(pod)
		podNode = pod.Spec.NodeName
	}

	var notes []string
	// check error states
	if overallNovaAttachmentCount >= 2 {
		notes = append(notes, "multiple attachments")
	}
	if podNode != "-" && podStatus != "Completed" && !strings.Contains(strings.Join(cinderServers, " "), podNode) {
		notes = append(notes, "pod != cinder server")
	}
	if podNode != "-" && podStatus != "Completed" && !strings.Contains(strings.Join(novaServers, " "), podNode) {
		notes = append(notes, "pod != nova server")
	}
	if !strings.Contains(strings.Join(novaServers, " "), strings.Join(cinderServers, " ")) {
		notes = append(notes, "nova != cinder server")
	}
	if v.Status == "available" && (len(novaServers) > 0 || len(cinderServers) > 0) {
		notes = append(notes, "available but attached")
	}
	if v.Status == "available" && podName != "-" && podStatus != "Completed" {
		notes = append(notes, fmt.Sprintf("available but pod %q", podStatus))
	}
	if v.Status == "in-use" && (len(novaServers) == 0 || len(cinderServers) == 0) {
		notes = append(notes, "in-use but not attached")
	}
	if strings.Contains(strings.Join(cinderServers, " "), "not found") {
		notes = append(notes, "attached server not found")
	}
	if pvClaim == "-" && pvName == "-" && podName == "-" && strings.HasPrefix(v.Name, "kubernetes-dynamic-pvc") {
		notes = append(notes, "kubernetes disk has no pv/pvc/pod")
	}
	note := strings.Join(notes, ", ")

	lineAllColumns := map[string]string{}
	lineAllColumns["CLUSTER"] = context
	lineAllColumns["PVC"] = pvClaim
	lineAllColumns["PV"] = pvName
	lineAllColumns["POD"] = podName
	lineAllColumns["POD_NODE"] = podNode
	lineAllColumns["POD_STATUS"] = podStatus
	lineAllColumns["CINDER_NAME"] = v.Name
	lineAllColumns["SIZE"] = fmt.Sprintf("%d", v.Size)
	lineAllColumns["CINDER_ID"] = v.ID
	lineAllColumns["CINDER_SERVER"] = strings.Join(cinderServers, " ")
	lineAllColumns["CINDER_SERVER_ID"] = strings.Join(cinderServerIDs, " ")
	lineAllColumns["CINDER_STATUS"] = v.Status
	lineAllColumns["NOVA_SERVER"] = strings.Join(novaServers, " ")
	lineAllColumns["NOVA_SERVER_ID"] = strings.Join(novaServerIDs, " ")
	lineAllColumns["NOTE"] = note

	return lineAllColumns
}
