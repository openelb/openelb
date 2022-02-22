package validate

import (
	"github.com/openelb/openelb/pkg/constant"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"strings"
)

func HasOpenELBAnnotation(annotation map[string]string) bool {
	if annotation == nil {
		return false
	}
	if value, ok := annotation[constant.OpenELBAnnotationKey]; ok {
		if value == constant.OpenELBAnnotationValue {
			return true
		}
	}
	return false
}

func HasOpenELBNPAnnotation(annotation map[string]string) bool {
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

func HasOpenELBDefaultEipAnnotation(annotation map[string]string) bool {
	if annotation == nil {
		return false
	}
	if value, ok := annotation[constant.OpenELBEIPAnnotationDefaultPool]; ok {
		return strings.ToLower(value) == "true"
	}
	return false
}
