package validate

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	PorterAnnotationKey   = "lb.kubesphere.io/v1alpha1"
	PorterAnnotationValue = "porter"
)

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
