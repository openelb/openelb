package speaker

import (
	"github.com/openelb/openelb/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
)

type Speaker interface {
	SetBalancer(ip string, nexthops []corev1.Node) error
	DelBalancer(ip string) error
	Start(stopCh <-chan struct{}) error
	ConfigureWithEIP(eip *v1alpha2.Eip, deleted bool) error
}
