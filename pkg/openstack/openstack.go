package openstack

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/listeners"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/monitors"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/pools"
	"gopkg.in/yaml.v2"
)

func GetVolumes(osProvider *gophercloud.ProviderClient) (map[string]volumes.Volume, error) {
	blockStorageClient, err := openstack.NewBlockStorageV3(osProvider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("error creating volume client: %v", err)
	}
	pager, err := volumes.List(blockStorageClient, volumes.ListOpts{}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error pageing volumes: %v", err)
	}
	vs, err := volumes.ExtractVolumes(pager)
	if err != nil {
		return nil, fmt.Errorf("error extracting volumes: %v", err)
	}
	volumeMap := map[string]volumes.Volume{}
	for _, v := range vs {
		volumeMap[v.ID] = v
	}
	return volumeMap, nil
}

func GetServer(osProvider *gophercloud.ProviderClient) (map[string]servers.Server, error) {
	computeClient, err := openstack.NewComputeV2(osProvider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("error creating compute client: %v", err)
	}
	pager, err := servers.List(computeClient, servers.ListOpts{}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("error pageing server: %v", err)
	}
	srvs, err := servers.ExtractServers(pager)
	if err != nil {
		return nil, fmt.Errorf("error extracting server: %v", err)
	}
	serverMap := map[string]servers.Server{}
	for _, srv := range srvs {
		serverMap[srv.ID] = srv
	}
	return serverMap, nil
}

func GetLB(osProvider *gophercloud.ProviderClient) (map[string]loadbalancers.LoadBalancer, map[string]listeners.Listener, map[string]pools.Pool, map[string]pools.Member, map[string]monitors.Monitor, map[string]floatingips.FloatingIP, error) {
	networkClient, err := openstack.NewNetworkV2(osProvider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error creating network client: %v", err)
	}

	pager, err := loadbalancers.List(networkClient, loadbalancers.ListOpts{}).AllPages()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error pageing loadbalancers: %v", err)
	}
	lbs, err := loadbalancers.ExtractLoadBalancers(pager)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error extracting loadbalancers: %v", err)
	}
	loadBalancersMap := map[string]loadbalancers.LoadBalancer{}
	for _, lb := range lbs {
		loadBalancersMap[lb.ID] = lb
	}

	pager, err = listeners.List(networkClient, listeners.ListOpts{}).AllPages()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error pageing listeners: %v", err)
	}
	ls, err := listeners.ExtractListeners(pager)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error extracting listeners: %v", err)
	}
	listenersMap := map[string]listeners.Listener{}
	for _, l := range ls {
		listenersMap[l.ID] = l
	}

	pager, err = pools.List(networkClient, pools.ListOpts{}).AllPages()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error pageing pools: %v", err)
	}
	poolss, err := pools.ExtractPools(pager)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error extracting pools: %v", err)
	}
	poolsMap := map[string]pools.Pool{}
	for _, p := range poolss {
		poolsMap[p.ID] = p
	}

	membersMap := map[string]pools.Member{}
	for _, pool := range poolss {
		pager, err = pools.ListMembers(networkClient, pool.ID, pools.ListMembersOpts{}).AllPages()
		if err != nil {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("error pageing lbmembers: %v", err)
		}
		members, err := pools.ExtractMembers(pager)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("error extracting lbmembers: %v", err)
		}
		for _, m := range members {
			m.PoolID = pool.ID
			membersMap[m.ID] = m
		}
	}

	pager, err = monitors.List(networkClient, monitors.ListOpts{}).AllPages()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error pageing monitors: %v", err)
	}
	monitorss, err := monitors.ExtractMonitors(pager)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error extracting monitors: %v", err)
	}
	monitorsMap := map[string]monitors.Monitor{}
	for _, m := range monitorss {
		monitorsMap[m.ID] = m
	}

	pager, err = floatingips.List(networkClient, floatingips.ListOpts{}).AllPages()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error pageing floatingips: %v", err)
	}
	floatingipss, err := floatingips.ExtractFloatingIPs(pager)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("error extracting floatingips: %v", err)
	}
	floatingipsMap := map[string]floatingips.FloatingIP{}
	for _, f := range floatingipss {
		floatingipsMap[f.ID] = f
	}

	return loadBalancersMap, listenersMap, poolsMap, membersMap, monitorsMap, floatingipsMap, nil
}

func GetOpenStackClient(context string) (*gophercloud.ProviderClient, string, error) {
	providerClient, tenantID, err := createOpenStackProviderClient(context)
	if err != nil {
		return nil, tenantID, fmt.Errorf("error creating openstack client: %v", err)
	}

	return providerClient, tenantID, nil
}

func createOpenStackProviderClient(context string) (*gophercloud.ProviderClient, string, error) {

	tenantID := strings.Split(context, "-")[0]

	openstackConfigFile := os.Getenv("OPENSTACK_CONFIG_FILE")
	if openstackConfigFile != "" {
		authOptions, err := getAuthOptionsFromConfig(openstackConfigFile, tenantID)
		if err != nil {
			return nil, tenantID, fmt.Errorf("error getting auth options from config file %s: %v", openstackConfigFile, err)
		}
		osProvider, err := openstack.AuthenticatedClient(*authOptions)
		return osProvider, tenantID, err
	}

	authOptions, err := getAuthOptionsFromEnv()
	if err != nil {
		return nil, tenantID, fmt.Errorf("error getting auth options from env variables: %v", err)
	}
	osProvider, err := openstack.AuthenticatedClient(*authOptions)
	return osProvider, tenantID, err
}

func getAuthOptionsFromEnv() (*gophercloud.AuthOptions, error) {
	authOptions := &gophercloud.AuthOptions{}

	authOptions.Username = os.Getenv("OS_USERNAME")
	if authOptions.Username == "" {
		return nil, fmt.Errorf("could not get username from env var OS_USERNAME")
	}
	authOptions.Password = os.Getenv("OS_PASSWORD")
	if authOptions.Password == "" {
		return nil, fmt.Errorf("could not get password from env var OS_PASSWORD")
	}
	authOptions.TenantName = os.Getenv("OS_PROJECT_NAME")
	if authOptions.TenantName == "" {
		authOptions.TenantName = os.Getenv("OS_TENANT_NAME")
	}
	authOptions.TenantID = os.Getenv("OS_PROJECT_ID")
	if authOptions.TenantID == "" {
		authOptions.TenantID = os.Getenv("OS_TENANT_ID")
	}

	if authOptions.TenantName == "" || authOptions.TenantID == "" {
		return nil, fmt.Errorf("could not get projectName or projectID from env vars OS_PROJECT_NAME, OS_TENANT_NAME, OS_PROJECT_ID or OS_TENANT_ID")
	}
	authOptions.IdentityEndpoint = os.Getenv("OS_AUTH_URL")
	if authOptions.IdentityEndpoint == "" {
		return nil, fmt.Errorf("could not get authUrl from env var OS_AUTH_URL")
	}
	return authOptions, nil
}

type clouds struct {
	Clouds map[string]cloud `yaml:"clouds"`
}
type cloud struct {
	Auth   cloudAuth `yaml:"auth"`
	Verify bool      `yaml:"verify"`
	CaCert string    `yaml:"cacert,omitempty"`
}

type cloudAuth struct {
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	DomainName  string `yaml:"domain_name"`
	AuthUrl     string `yaml:"auth_url"`
	ProjectName string `yaml:"project_name"`
	ProjectID   string `yaml:"project_id"`
}

//clouds:
//	devstack:
//		auth:
//			auth_url: http://192.168.122.10:35357/
//			project_name: demo
//			username: demo
//			password: 0penstack
// See https://docs.openstack.org/python-openstackclient/pike/configuration/index.html
func getAuthOptionsFromConfig(configFile, context string) (*gophercloud.AuthOptions, error) {
	content, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("error reading config file %s: %v", configFile, err)
	}

	var config clouds
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file %s: %v", configFile, err)
	}

	cloud, ok := config.Clouds[context]
	if !ok {
		return nil, fmt.Errorf("could not find cloud %s in config file %s", context, configFile)
	}

	if cloud.Auth.AuthUrl == "" {
		return nil, fmt.Errorf("could not find auth_url in cloud %s in config file %s", context, configFile)
	}

	projectName := cloud.Auth.ProjectName
	projectID := cloud.Auth.ProjectID
	if projectName == "" && projectID == "" {
		return nil, fmt.Errorf("could not find project_name or project_id in cloud %s in config file %s", context, configFile)
	}

	if cloud.Auth.Username == "" {
		return nil, fmt.Errorf("could not find username in cloud %s in config file %s", context, configFile)
	}
	if cloud.Auth.Password == "" {
		return nil, fmt.Errorf("could not find password in cloud %s in config file %s", context, configFile)
	}
	options := gophercloud.AuthOptions{IdentityEndpoint: cloud.Auth.AuthUrl, Username: cloud.Auth.Username, Password: cloud.Auth.Password}
	if projectName != "" {
		options.TenantName = projectName
	}
	if projectID != "" {
		options.TenantID = projectID
	}
	options.DomainName = cloud.Auth.DomainName
	return &options, nil
}

func GetVolumeAttachmentsForServerNova(osProvider *gophercloud.ProviderClient, servers map[string]servers.Server) (map[string]*NovaVolumeAttachments, error) {
	attachments := map[string]*NovaVolumeAttachments{}

	for server := range servers {
		volumeAttachments, err := GetVolumeAttachmentsNova(osProvider, server)
		if err != nil {
			return nil, err
		}
		attachments[server] = volumeAttachments
	}

	return attachments, nil
}

func GetVolumeAttachmentsNova(osProvider *gophercloud.ProviderClient, serverID string) (*NovaVolumeAttachments, error) {

	computeClient, err := openstack.NewComputeV2(osProvider, gophercloud.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("error creating compute client: %v", err)
	}

	url := computeClient.ServiceURL("servers", serverID, "os-volume_attachments")

	attachments := &NovaVolumeAttachments{}
	_, err = computeClient.Get(url, attachments, &gophercloud.RequestOpts{OkCodes: []int{200}})
	if err != nil {
		return nil, fmt.Errorf("error attaching volume: %v", err)
	}

	return attachments, nil
}

func AttachVolumeNova(osProvider *gophercloud.ProviderClient, volumeID, serverID string) error {

	fmt.Printf("Attaching volume %s to server %s via nova\n", volumeID, serverID)

	computeClient, err := openstack.NewComputeV2(osProvider, gophercloud.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("error creating compute client: %v", err)
	}

	url := computeClient.ServiceURL("servers", serverID, "os-volume_attachments")

	attach := &novaAttachVolume{
		VolumeAttachment: &novaAttachment{
			VolumeID: volumeID,
		},
	}

	resp, err := computeClient.Post(url, attach, nil, &gophercloud.RequestOpts{OkCodes: []int{200}})
	if err != nil {
		return fmt.Errorf("error attaching volume: %v", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response from attach volume: %v", err)
	}

	fmt.Printf("Response from attach volume from nova: %s\n", string(body))

	return nil
}

func DetachVolumeNova(osProvider *gophercloud.ProviderClient, volumeID, serverID string) error {

	fmt.Printf("Detaching volume %s from server %s via nova\n", volumeID, serverID)

	computeClient, err := openstack.NewComputeV2(osProvider, gophercloud.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("error creating compute client: %v", err)
	}

	url := computeClient.ServiceURL("servers", serverID, "os-volume_attachments", volumeID)

	resp, err := computeClient.Delete(url, &gophercloud.RequestOpts{OkCodes: []int{202}})
	if err != nil {
		return fmt.Errorf("error deleting volume from nova: %v", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response from detach volume: %v", err)
	}

	fmt.Printf("Response from detach volume from nova: %s\n", string(body))

	return nil
}

type NovaVolumeAttachments struct {
	VolumeAttachments []*novaAttachment `json:"volumeAttachments"`
}

type novaAttachVolume struct {
	VolumeAttachment *novaAttachment `json:"volumeAttachment"`
}

type novaAttachment struct {
	ID       string ` json:"id,omitempty"`
	Device   string ` json:"device,omitempty"`
	ServerID string ` json:"serverId,omitempty"`
	VolumeID string ` json:"volumeId,omitempty"`
}

func DetachVolumeCinder(osProvider *gophercloud.ProviderClient, volumeID string, force bool) error {

	fmt.Printf("Detaching volume %s from cinder (force: %t)\n", volumeID, force)

	blockStorageClient, err := openstack.NewBlockStorageV3(osProvider, gophercloud.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("error creating volume client: %v", err)
	}

	url := blockStorageClient.ServiceURL("volumes", volumeID, "action")

	var detach interface{}
	if force {
		detach = &cinderForceDetachVolume{
			OsDetach: &cinderDetachment{AttachmentID: volumeID},
		}
	} else {
		detach = &cinderDetachVolume{
			OsDetach: &cinderDetachment{AttachmentID: volumeID},
		}
	}
	resp, err := blockStorageClient.Post(url, detach, nil, &gophercloud.RequestOpts{OkCodes: []int{202}})
	if err != nil {
		return fmt.Errorf("error deleting volume from cinder: %v", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response from detach volume: %v", err)
	}

	fmt.Printf("Response from detach volume from cinder: %s\n", string(body))

	return nil
}

func AttachVolumeCinder(osProvider *gophercloud.ProviderClient, volumeID, serverID, mountpoint string) error {

	fmt.Printf("Attaching volume %s to server %s via cinder\n", volumeID, serverID)

	blockStorageClient, err := openstack.NewBlockStorageV3(osProvider, gophercloud.EndpointOpts{})
	if err != nil {
		return fmt.Errorf("error creating volume client: %v", err)
	}

	url := blockStorageClient.ServiceURL("volumes", volumeID, "action")

	attach := &cinderAttachVolume{
		OsAttach: &cinderAttachment{
			InstanceUUID: serverID,
			Mountpoint:   mountpoint,
		},
	}

	resp, err := blockStorageClient.Post(url, attach, nil, &gophercloud.RequestOpts{OkCodes: []int{202}})
	if err != nil {
		return fmt.Errorf("error attaching volume: %v", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response from attach volume: %v", err)
	}

	fmt.Printf("Response from attach volume from cinder: %s\n", string(body))

	return nil
}

type cinderForceDetachVolume struct {
	OsDetach *cinderDetachment `json:"os-force_detach"`
}

type cinderDetachVolume struct {
	OsDetach *cinderDetachment `json:"os-detach"`
}

type cinderDetachment struct {
	AttachmentID string ` json:"attachment_id"`
}

type cinderAttachVolume struct {
	OsAttach *cinderAttachment `json:"os-attach"`
}

type cinderAttachment struct {
	InstanceUUID string ` json:"instance_uuid"`
	Mountpoint   string ` json:"mountpoint"`
}
