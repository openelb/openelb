package validate

import (
	"github.com/kubesphere/porterlb/pkg/constant"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func HasPorterLBAnnotation(annotation map[string]string) bool {
	if annotation == nil {
		return false
	}
	if value, ok := annotation[constant.PorterAnnotationKey]; ok {
		if value == constant.PorterAnnotationValue {
			return true
		}
	}
	return false
}

func HasPorterNPAnnotation(annotation map[string]string) bool {
	if annotation == nil {
		return false
	}
	if value, ok := annotation[constant.NodeProxyTypeAnnotationKey]; ok {
		if value == constant.NodeProxyTypeDeployment || value == constant.NodeProxyTypeDaemonSet {
			return true
		}
	}
	return false
}

func IsTypeLoadBalancer(obj runtime.Object) bool {
	if svc, ok := obj.(*corev1.Service); ok {
		return svc.Spec.Type == corev1.ServiceTypeLoadBalancer
	}
	return false
}
