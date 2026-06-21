package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Namespace struct {
	Name   string
	Labels map[string]string
}

type Deployment struct {
	Name      string
	Namespace string
	Replicas  int32
	Ready     int32
	Labels    map[string]string
	Selector  map[string]string
}

type ReplicaSet struct {
	Name      string
	Namespace string
	OwnerName string
	Replicas  int32
	Ready     int32
}

type Pod struct {
	Name       string
	Namespace  string
	OwnerName  string
	NodeName   string
	Phase      string
	IP         string
	Labels     map[string]string
	Containers []Container
}

type Container struct {
	Name  string
	Image string
	Ready bool
}

type Node struct {
	Name      string
	Labels    map[string]string
	Addresses []string
	Capacity  ResourceList
	Allocated ResourceList
}

type ResourceList struct {
	CPU    string
	Memory string
}

type Service struct {
	Name      string
	Namespace string
	Type      string
	Selector  map[string]string
	Ports     []ServicePort
	ClusterIP string
}

type ServicePort struct {
	Name       string
	Port       int32
	TargetPort int32
	Protocol   string
}

type NetworkPolicy struct {
	Name        string
	Namespace   string
	PodSelector map[string]string
	Ingress     []NetworkPolicyRule
	Egress      []NetworkPolicyRule
}

type NetworkPolicyRule struct {
	Ports []NetworkPolicyPort
	From  []NetworkPolicyPeer
	To    []NetworkPolicyPeer
}

type NetworkPolicyPort struct {
	Port     int32
	Protocol string
}

type NetworkPolicyPeer struct {
	PodSelector       map[string]string
	NamespaceSelector map[string]string
	IPBlock           string
}

type Endpoint struct {
	Name      string
	Namespace string
	Addresses []EndpointAddress
	Ports     []EndpointPort
}

type EndpointAddress struct {
	IP       string
	NodeName string
	PodName  string
}

type EndpointPort struct {
	Port     int32
	Protocol string
}

type K8sFlow struct {
	SrcPod       string
	SrcNamespace string
	DstPod       string
	DstNamespace string
	DstIP        string
	DstPort      int32
	Protocol     string
	Allowed      bool
	PolicyName   string
}

type TopologySnapshot struct {
	Namespaces      []Namespace
	Deployments     []Deployment
	ReplicaSets     []ReplicaSet
	Pods            []Pod
	Nodes           []Node
	Services        []Service
	NetworkPolicies []NetworkPolicy
	Endpoints       []Endpoint
	Flows           []K8sFlow
}

type Collector struct {
	clientset *kubernetes.Clientset
	logger    *slog.Logger
}

func NewCollector(logger *slog.Logger) (*Collector, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("getting kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating clientset: %w", err)
	}

	return &Collector{
		clientset: clientset,
		logger:    logger,
	}, nil
}

func getKubeConfig() (*rest.Config, error) {
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		return rest.InClusterConfig()
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func (c *Collector) Collect(ctx context.Context) (*TopologySnapshot, error) {
	snapshot := &TopologySnapshot{}

	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing namespaces: %w", err)
	}

	for _, ns := range namespaces.Items {
		snapshot.Namespaces = append(snapshot.Namespaces, Namespace{
			Name:   ns.Name,
			Labels: ns.Labels,
		})
	}

	deployments, err := c.clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing deployments: %w", err)
	}

	for _, dep := range deployments.Items {
		snapshot.Deployments = append(snapshot.Deployments, convertDeployment(&dep))
	}

	replicasets, err := c.clientset.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing replicasets: %w", err)
	}

	for _, rs := range replicasets.Items {
		snapshot.ReplicaSets = append(snapshot.ReplicaSets, convertReplicaSet(&rs))
	}

	pods, err := c.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	for _, pod := range pods.Items {
		snapshot.Pods = append(snapshot.Pods, convertPod(&pod))
	}

	nodes, err := c.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	for _, node := range nodes.Items {
		snapshot.Nodes = append(snapshot.Nodes, convertNode(&node))
	}

	services, err := c.clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	for _, svc := range services.Items {
		snapshot.Services = append(snapshot.Services, convertService(&svc))
	}

	networkPolicies, err := c.clientset.NetworkingV1().NetworkPolicies("").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("listing network policies", "error", err)
	} else {
		for _, np := range networkPolicies.Items {
			snapshot.NetworkPolicies = append(snapshot.NetworkPolicies, convertNetworkPolicy(&np))
		}
	}

	endpoints, err := c.clientset.CoreV1().Endpoints("").List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("listing endpoints", "error", err)
	} else {
		for _, ep := range endpoints.Items {
			snapshot.Endpoints = append(snapshot.Endpoints, convertEndpoints(&ep))
		}
	}

	snapshot.Flows = c.deriveFlows(snapshot.Services, snapshot.Pods, snapshot.NetworkPolicies, snapshot.Endpoints)

	c.logger.Info("k8s topology collected",
		"namespaces", len(snapshot.Namespaces),
		"deployments", len(snapshot.Deployments),
		"pods", len(snapshot.Pods),
		"nodes", len(snapshot.Nodes),
		"services", len(snapshot.Services),
	)

	return snapshot, nil
}

func convertDeployment(dep *appsv1.Deployment) Deployment {
	var replicas, ready int32
	if dep.Spec.Replicas != nil {
		replicas = *dep.Spec.Replicas
	}
	ready = dep.Status.ReadyReplicas

	return Deployment{
		Name:      dep.Name,
		Namespace: dep.Namespace,
		Replicas:  replicas,
		Ready:     ready,
		Labels:    dep.Labels,
		Selector:  dep.Spec.Selector.MatchLabels,
	}
}

func convertReplicaSet(rs *appsv1.ReplicaSet) ReplicaSet {
	var replicas, ready int32
	if rs.Spec.Replicas != nil {
		replicas = *rs.Spec.Replicas
	}
	ready = rs.Status.ReadyReplicas

	ownerName := ""
	if len(rs.OwnerReferences) > 0 {
		ownerName = rs.OwnerReferences[0].Name
	}

	return ReplicaSet{
		Name:      rs.Name,
		Namespace: rs.Namespace,
		OwnerName: ownerName,
		Replicas:  replicas,
		Ready:     ready,
	}
}

func convertPod(pod *corev1.Pod) Pod {
	var containers []Container
	for _, c := range pod.Spec.Containers {
		ready := false
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Name == c.Name {
				ready = cs.Ready
				break
			}
		}
		containers = append(containers, Container{
			Name:  c.Name,
			Image: c.Image,
			Ready: ready,
		})
	}

	ownerName := ""
	if len(pod.OwnerReferences) > 0 {
		ownerName = pod.OwnerReferences[0].Name
	}

	return Pod{
		Name:       pod.Name,
		Namespace:  pod.Namespace,
		OwnerName:  ownerName,
		NodeName:   pod.Spec.NodeName,
		Phase:      string(pod.Status.Phase),
		IP:         pod.Status.PodIP,
		Labels:     pod.Labels,
		Containers: containers,
	}
}

func convertNode(node *corev1.Node) Node {
	addresses := make([]string, 0)
	for _, addr := range node.Status.Addresses {
		addresses = append(addresses, addr.Address)
	}

	return Node{
		Name:      node.Name,
		Labels:    node.Labels,
		Addresses: addresses,
		Capacity: ResourceList{
			CPU:    node.Status.Capacity.Cpu().String(),
			Memory: node.Status.Capacity.Memory().String(),
		},
		Allocated: ResourceList{
			CPU:    node.Status.Allocatable.Cpu().String(),
			Memory: node.Status.Allocatable.Memory().String(),
		},
	}
}

func convertService(svc *corev1.Service) Service {
	var ports []ServicePort
	for _, p := range svc.Spec.Ports {
		ports = append(ports, ServicePort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: p.TargetPort.IntVal,
			Protocol:   string(p.Protocol),
		})
	}

	return Service{
		Name:      svc.Name,
		Namespace: svc.Namespace,
		Type:      string(svc.Spec.Type),
		Selector:  svc.Spec.Selector,
		Ports:     ports,
		ClusterIP: svc.Spec.ClusterIP,
	}
}

func convertNetworkPolicy(np *networkingv1.NetworkPolicy) NetworkPolicy {
	var ingress []NetworkPolicyRule
	for _, rule := range np.Spec.Ingress {
		r := NetworkPolicyRule{}
		for _, port := range rule.Ports {
			r.Ports = append(r.Ports, NetworkPolicyPort{
				Port:     port.Port.IntVal,
				Protocol: string(*port.Protocol),
			})
		}
		for _, peer := range rule.From {
			p := NetworkPolicyPeer{}
			if peer.PodSelector != nil {
				p.PodSelector = peer.PodSelector.MatchLabels
			}
			if peer.NamespaceSelector != nil {
				p.NamespaceSelector = peer.NamespaceSelector.MatchLabels
			}
			if peer.IPBlock != nil {
				p.IPBlock = peer.IPBlock.CIDR
			}
			r.From = append(r.From, p)
		}
		ingress = append(ingress, r)
	}

	var egress []NetworkPolicyRule
	for _, rule := range np.Spec.Egress {
		r := NetworkPolicyRule{}
		for _, port := range rule.Ports {
			r.Ports = append(r.Ports, NetworkPolicyPort{
				Port:     port.Port.IntVal,
				Protocol: string(*port.Protocol),
			})
		}
		for _, peer := range rule.To {
			p := NetworkPolicyPeer{}
			if peer.PodSelector != nil {
				p.PodSelector = peer.PodSelector.MatchLabels
			}
			if peer.NamespaceSelector != nil {
				p.NamespaceSelector = peer.NamespaceSelector.MatchLabels
			}
			if peer.IPBlock != nil {
				p.IPBlock = peer.IPBlock.CIDR
			}
			r.To = append(r.To, p)
		}
		egress = append(egress, r)
	}

	return NetworkPolicy{
		Name:        np.Name,
		Namespace:   np.Namespace,
		PodSelector: np.Spec.PodSelector.MatchLabels,
		Ingress:     ingress,
		Egress:      egress,
	}
}

func convertEndpoints(ep *corev1.Endpoints) Endpoint {
	var addresses []EndpointAddress
	for _, subset := range ep.Subsets {
		for _, addr := range subset.Addresses {
			addresses = append(addresses, EndpointAddress{
				IP: addr.IP,
				NodeName: func() string {
					if addr.NodeName != nil {
						return *addr.NodeName
					}
					return ""
				}(),
				PodName: func() string {
					if addr.TargetRef != nil {
						return addr.TargetRef.Name
					}
					return ""
				}(),
			})
		}
	}

	var ports []EndpointPort
	for _, subset := range ep.Subsets {
		for _, port := range subset.Ports {
			ports = append(ports, EndpointPort{
				Port:     port.Port,
				Protocol: string(port.Protocol),
			})
		}
	}

	return Endpoint{
		Name:      ep.Name,
		Namespace: ep.Namespace,
		Addresses: addresses,
		Ports:     ports,
	}
}

func (c *Collector) deriveFlows(services []Service, pods []Pod, policies []NetworkPolicy, endpoints []Endpoint) []K8sFlow {
	var flows []K8sFlow

	for _, svc := range services {
		for _, ep := range endpoints {
			if ep.Name == svc.Name && ep.Namespace == svc.Namespace {
				for _, addr := range ep.Addresses {
					for _, port := range svc.Ports {
						flow := K8sFlow{
							SrcPod:       "service:" + svc.Name,
							SrcNamespace: svc.Namespace,
							DstPod:       addr.PodName,
							DstNamespace: svc.Namespace,
							DstIP:        addr.IP,
							DstPort:      port.Port,
							Protocol:     port.Protocol,
							Allowed:      true,
						}
						flows = append(flows, flow)
					}
				}
			}
		}
	}

	return flows
}
