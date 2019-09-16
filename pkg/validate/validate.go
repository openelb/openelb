package validate

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	PorterAnnotationKey   = "lb.kubesphere.io/v1alpha1"
	PorterAnnotationValue = "porter"
)

func IsPorterService(svc *corev1.Service) bool {
	return HasPorterLBAnnotation(svc.Annotations) && IsTypeLoadBalancer(svc)
}

func HasPorterLBAnnotation(annotation map[string]string) bool {
	if annotation == nil {
		return false
	}
	if value, ok := annotation[PorterAnnotationKey]; ok {
		if value == PorterAnnotationValue {
			return true
		}
	}
	return false
}

func IsTypeLoadBalancer(obj runtime.Object) bool {
	if ser, ok := obj.(*corev1.Service); ok {
		return ser.Spec.Type == corev1.ServiceTypeLoadBalancer
	}
	return false
}

func IsNodeChangedWhenEndpointUpdated(a *corev1.Endpoints, b *corev1.Endpoints) bool {
	if len(a.Subsets) != len(b.Subsets) {
		return true
	}
	if len(a.Subsets) == 0 {
		return false
	}
	if (len(a.Subsets[0].Addresses) + len(a.Subsets[0].NotReadyAddresses)) != (len(b.Subsets[0].Addresses) + len(b.Subsets[0].NotReadyAddresses)) {
		return true
	}
	nodeMapa := make(map[string]bool)
	for _, addr := range a.Subsets[0].Addresses {
		nodeMapa[*addr.NodeName] = true
	}
	for _, addr := range a.Subsets[0].NotReadyAddresses {
		nodeMapa[*addr.NodeName] = true
	}
	nodeMapb := make(map[string]interface{})
	for _, addr := range b.Subsets[0].Addresses {
		nodeMapb[*addr.NodeName] = true
	}
	for _, addr := range b.Subsets[0].NotReadyAddresses {
		nodeMapb[*addr.NodeName] = true
	}
	if len(nodeMapa) != len(nodeMapb) {
		return true
	}
	for key := range nodeMapa {
		if _, ok := nodeMapb[key]; !ok {
			return true
		}
	}
	return false
}
