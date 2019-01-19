package kubernetes

import (
	"fmt"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	"regexp"
	"sort"
	"strings"
)

func GetKubeClient(config *rest.Config) (*kubernetes.Clientset, error) {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating kube client: %v", err)
	}

	return clientSet, nil
}

func GetNodes(kubeClient *kubernetes.Clientset) (map[string]v1.Node, error) {
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

func GetServices(kubeClient *kubernetes.Clientset) (map[int32]v1.Service, error) {
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

func GetPersistentVolumes(kubeClient *kubernetes.Clientset) (map[string]v1.PersistentVolume, error) {
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

func GetPodsByPVC(kubeClient *kubernetes.Clientset) (map[string]v1.Pod, error) {
	pods, err := kubeClient.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting pods: %v", err)
	}
	podMap := map[string]v1.Pod{}
	for _, pod := range pods.Items {
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName != "" {
				pvcName := fmt.Sprintf("%s/%s", pod.Namespace, volume.PersistentVolumeClaim.ClaimName)
				podMap[pvcName] = pod
			}
		}
	}
	return podMap, nil
}

func GetMatchingContexts(config api.Config, context string) []string {
	if context == "" {
		return []string{config.CurrentContext}
	}

	matchingContexts := map[string]bool{}
	for _, contextRegexp := range strings.Split(context, ",") {
		regex, err := regexp.Compile(contextRegexp)
		if err != nil {
			fmt.Printf("Regex %s does not compile: %v\n", contextRegexp, err)
		}
		for context := range config.Contexts {
			if regex.Match([]byte(context)) {
				matchingContexts[context] = true
			}
		}
	}
	var ctxs []string
	for ctx, ok := range matchingContexts {
		if ok {
			ctxs = append(ctxs, ctx)
		}
	}
	sort.Strings(ctxs)
	return ctxs
}


func GetPodStatus(pod v1.Pod) string {
	if pod.Status.Phase == v1.PodFailed && pod.Status.Reason == "Evicted" {
		return "Evicted"
	}
	for _, c := range pod.Status.Conditions {
		if c.Status == v1.ConditionFalse && c.Type == v1.PodReady && c.Reason == "PodCompleted" {
			return "Completed"
		}
		if c.Status == v1.ConditionTrue && c.Type == v1.PodReady {
			return "Running"
		}
		if c.Status == v1.ConditionFalse && c.Type == v1.PodReady && c.Reason == "ContainersNotReady" {
			for _, c := range pod.Status.ContainerStatuses {
				if c.State.Waiting != nil {
					if c.State.Waiting.Reason == "ImagePullBackOff" {
						return c.State.Waiting.Reason
					}
					if c.State.Waiting.Reason == "ContainerCreating" {
						return c.State.Waiting.Reason
					}
					if c.State.Waiting.Reason == "CreateContainerConfigError" {
						return c.State.Waiting.Reason
					}
				}
			}
		}
	}
	return "Not Implemented yet"
}
