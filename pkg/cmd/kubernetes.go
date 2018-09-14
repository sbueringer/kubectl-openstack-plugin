package cmd

import (
	"fmt"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"strings"
)

func getKubeClient(flags *genericclioptions.ConfigFlags) (*kubernetes.Clientset, error) {
	config, err := flags.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("error parsing config flags: %v", err)
	}

	clientset, _ := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating kube client: %v", err)
	}

	return clientset, nil
}

func getNodes(kubeClient *kubernetes.Clientset) (map[string]v1.Node, error) {
	nodes, err := kubeClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting persistent volumes: %v", err)
	}
	nodesMap := map[string]v1.Node{}
	for _, node := range nodes.Items {
		osID := strings.TrimPrefix(node.Spec.ProviderID, "openstack:///")
		nodesMap[osID] = node
	}
	return nodesMap, nil
}

func getServices(kubeClient *kubernetes.Clientset) (map[int32]v1.Service, error) {
	services, err := kubeClient.CoreV1().Services("").List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting persistent volumes: %v", err)
	}
	servicesMap := map[int32]v1.Service{}
	for _, svc := range services.Items {
		for _, port := range svc.Spec.Ports {
			if port.NodePort != 0 {
				servicesMap[int32(port.NodePort)] = svc
			}
		}
	}
	return servicesMap, nil
}

func getPersistentVolumes(kubeClient *kubernetes.Clientset) (map[string]v1.PersistentVolume, error) {
	pvs, err := kubeClient.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting persistent volumes: %v", err)
	}
	pvMap := map[string]v1.PersistentVolume{}
	for _, pv := range pvs.Items {
		if pv.Spec.Cinder == nil {
			// TODO log(skipping pv because it is no cinder volume)
			continue
		}
		pvMap[pv.Spec.Cinder.VolumeID] = pv
	}
	return pvMap, nil
}
