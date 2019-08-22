package kubeutil

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetServiceNodesIP(c client.Client, serv *corev1.Service) ([]string, error) {
	endpoints := &corev1.Endpoints{}
	err := c.Get(context.TODO(), types.NamespacedName{Namespace: serv.GetNamespace(), Name: serv.GetName()}, endpoints)
	if err != nil {
		return nil, err
	}
	if len(endpoints.Subsets) == 0 {
		return nil, nil
	}
	nodes := make(map[string]bool)
	for _, addr := range endpoints.Subsets[0].Addresses {
		nodes[*addr.NodeName] = true
	}
	for _, addr := range endpoints.Subsets[0].NotReadyAddresses {
		nodes[*addr.NodeName] = true
	}
	nodeIPMap, err := GetNodeIPMap(c)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0)
	for key := range nodes {
		result = append(result, nodeIPMap[key])
	}
	return result, nil
}

func GetNodeIPMap(c client.Client) (map[string]string, error) {
	nodeList := &corev1.NodeList{}
	err := c.List(context.TODO(), nodeList)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, node := range nodeList.Items {
		result[node.Name] = node.Status.Addresses[0].Address
	}
	return result, nil
}
