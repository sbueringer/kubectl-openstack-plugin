package cmd

import (
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd/api"
	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"

	"fmt"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/listeners"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/pools"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/monitors"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/floatingips"
	"strings"
)

//TODO
type LBOptions struct {
	configFlags *genericclioptions.ConfigFlags
	rawConfig   api.Config
	//TODO decide what todo with list
	list bool
	args []string

	genericclioptions.IOStreams
}

var (
	lbExample = `
	# list lb
	%[1] lb
`
)

//TODO
func NewCmdLB(streams genericclioptions.IOStreams) *cobra.Command {
	o := &LBOptions{
		configFlags: genericclioptions.NewConfigFlags(),
		IOStreams:   streams,
	}
	cmd := &cobra.Command{
		Use: "lb",
		//Aliases:      []string{"lb"},
		Short:        "List all lb and corresponding services from Kubernetes and OpenStack",
		Example:      fmt.Sprintf(lbExample, "kubectl os"),
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
func (o *LBOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	var err error
	o.rawConfig, err = o.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}
	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *LBOptions) Validate() error {
	if len(o.rawConfig.CurrentContext) == 0 {
		return errNoContext
	}

	return nil
}

// Run lists all volumes
func (o *LBOptions) Run() error {

	fmt.Printf("%t\n", o.list)

	kubeClient, err := getKubeClient(o.configFlags)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}
	osProvider, err := getOpenStackClient(o.rawConfig)
	if err != nil {
		return fmt.Errorf("error creating client: %v", err)
	}

	servicesMap, err := getServices(kubeClient)
	if err != nil {
		return fmt.Errorf("error getting persistent volumes from Kubernetes: %v", err)
	}

	loadBalancersMap, listenersMap, poolsMap, membersMap, monitorsMap, floatingipsMap, err := getLB(osProvider)
	if err != nil {
		return fmt.Errorf("error getting servers from OpenStack: %v", err)
	}

	output, err := getPrettyLBList(servicesMap, loadBalancersMap, listenersMap, poolsMap, membersMap, monitorsMap, floatingipsMap)
	if err != nil {
		return fmt.Errorf("error creating ouput: %v", err)
	}
	fmt.Printf(output)

	return nil
}

func getPrettyLBList(services map[int32]v1.Service, loadbalancers map[string]loadbalancers.LoadBalancer, listeners map[string]listeners.Listener, pools map[string]pools.Pool, members map[string]pools.Member, monitors map[string]monitors.Monitor, floatingIPs map[string]floatingips.FloatingIP) (string, error) {

	header := []string{"NAME", "FLOATING_IPS", "VIP_ADDRESS", "PORTS", "SERVICES"}

	var lines [][]string
	poolsPerListener := getPoolsPerListener(pools)

	for _, lb := range loadbalancers {
		listenerss := getListener(lb.ID, listeners)

		floatingIPs := getFloatingIPForLB(lb, floatingIPs)
		floatingIPsString := strings.Join(floatingIPs, ",")

		for _, l := range listenerss {
			targets := map[int][]string{}
			poolss := poolsPerListener[l.ID]
			for _, pool := range poolss {
				for _, member := range getLBMemberForPool(pool, members) {
					ports, ok := targets[member.ProtocolPort]
					if !ok {
						ports = []string{}
					}
					targets[member.ProtocolPort] = append(ports, member.Address)
				}
			}
			var targetsArray []string
			var svcsArray []string
			for port, addresses := range targets {
				targetsArray = append(targetsArray, fmt.Sprintf("%s:%d", addresses, port))

				svc, ok := services[int32(port)]
				if ok {
					svcsArray = append(svcsArray, fmt.Sprintf("%s/%s", svc.Namespace, svc.Name))
				}
			}
			portMapping := fmt.Sprintf("%d => %s", l.ProtocolPort, strings.Join(targetsArray, ","))
			svcs := strings.Join(svcsArray, ",")

			lines = append(lines, []string{lb.Name, floatingIPsString, lb.VipAddress, portMapping, svcs})
		}
	}
	return printTable(table{header, lines, 0})
}
func getFloatingIPForLB(lb loadbalancers.LoadBalancer, floatingIPs map[string]floatingips.FloatingIP) []string {
	var fips []string
	for _, floatingIP := range floatingIPs {
		if floatingIP.PortID == lb.VipPortID {
			fips = append(fips, floatingIP.FloatingIP)
		}
	}
	return fips
}

func getLBMemberForPool(pool pools.Pool, members map[string]pools.Member) []pools.Member {
	var poolMember []pools.Member
	for _, member := range members {
		if member.PoolID == pool.ID {
			poolMember = append(poolMember, member)
		}
	}
	return poolMember
}
func getPoolsPerListener(poolss map[string]pools.Pool) map[string][]pools.Pool {
	poolsPerListener := map[string][]pools.Pool{}
	for _, pool := range poolss {
		for _, listenerID := range pool.Listeners {
			pl, ok := poolsPerListener[listenerID.ID]
			if !ok {
				pl = []pools.Pool{}
			}
			pl = append(pl, pool)
			poolsPerListener[listenerID.ID] = pl
		}
	}
	return poolsPerListener
}

func getListener(loadbalancerID string, listenerss map[string]listeners.Listener) []listeners.Listener {
	var ls []listeners.Listener
	for _, listener := range listenerss {
		for _, lb := range listener.Loadbalancers {
			if lb.ID == loadbalancerID {
				ls = append(ls, listener)
			}
		}
	}
	return ls
}
