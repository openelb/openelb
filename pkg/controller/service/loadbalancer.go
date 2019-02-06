package service

import (
	"context"
	"fmt"

	"github.com/kubesphere/porter/pkg/apis/network/v1alpha1"
	"github.com/kubesphere/porter/pkg/bgp/routes"
	"github.com/kubesphere/porter/pkg/strategy"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileService) getExternalIP(serv *corev1.Service) (string, error) {
	if len(serv.Spec.ExternalIPs) > 0 {
		return serv.Spec.ExternalIPs[0], nil
	}
	eipList := &v1alpha1.EIPList{}
	err := r.List(context.Background(), &client.ListOptions{}, eipList)
	if err != nil {
		return "", err
	}
	ipStrategy, _ := strategy.GetStrategy(strategy.DefaultStrategy)
	ip, err := ipStrategy.Select(serv, eipList)
	if err != nil {
		return "", err
	}
	return ip.Spec.Address, nil
}

func (r *ReconcileService) createLB(serv *corev1.Service) error {
	ip, err := r.getExternalIP(serv)
	if err != nil {
		return err
	}
	nexthops, err := r.getServiceNodesIP(serv)
	if err != nil {
		log.Error(nil, "Failed to get ip of nodes where endpoints locate in")
		return err
	}
	if err := routes.AddRoute(ip, nexthops); err != nil {
		return err
	}
	log.Info("Routed added successful", "ServiceName", serv.Name, "Namespace", serv.Namespace)
	err = r.markEIPPorts(ip, serv.Spec.Ports, true)
	if err != nil {
		log.Error(nil, "failed to mark ports of ip used")
		return err
	}
	log.Info(fmt.Sprintf("Pls visit %s:%d to check it out", ip, serv.Spec.Ports[0].Port))
	return nil
}

func (r *ReconcileService) deleteLB(serv *corev1.Service) error {
	ip, err := r.getExternalIP(serv)
	if err != nil {
		return err
	}
	nodeIPs, err := r.getServiceNodesIP(serv)
	if err != nil {
		log.Error(nil, "error in get nodes ip")
		return err
	}

	err = routes.DeleteRoutes(ip, nodeIPs)
	if err != nil {
		log.Error(nil, "Failed to delete routes")
	}
	log.Info("Routed deleted successful", "ServiceName", serv.Name, "Namespace", serv.Namespace)
	return nil
}

func (r *ReconcileService) checkLB(serv *corev1.Service) bool {
	ip, err := r.getExternalIP(serv)
	if err != nil {
		log.Error(err, "Failed to get ip")
		return false
	}
	return routes.IsRouteAdded(ip, 32)
}

func (r *ReconcileService) getServiceNodesIP(serv *corev1.Service) ([]string, error) {
	endpoints := &corev1.Endpoints{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: serv.GetNamespace(), Name: serv.GetName()}, endpoints)
	if err != nil {
		return nil, err
	}
	nodes := make(map[string]bool)
	for _, addr := range endpoints.Subsets[0].Addresses {
		nodes[*addr.NodeName] = true
	}
	for _, addr := range endpoints.Subsets[0].NotReadyAddresses {
		nodes[*addr.NodeName] = true
	}
	nodeIPMap, err := r.getNodeIPMap()
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)
	for key := range nodes {
		result = append(result, nodeIPMap[key])
	}
	return result, nil
}

func (r *ReconcileService) getNodeIPMap() (map[string]string, error) {
	nodeList := &corev1.NodeList{}
	err := r.List(context.TODO(), &client.ListOptions{}, nodeList)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, node := range nodeList.Items {
		result[node.Name] = node.Status.Addresses[0].Address
	}
	return result, nil
}

func (r *ReconcileService) markEIPPorts(ip string, ports []corev1.ServicePort, used bool) error {
	return nil
}
