package speaker

import (
	"github.com/openelb/openelb/pkg/util/iprange"
	corev1 "k8s.io/api/core/v1"
)

type Config struct {
	Name    string
	IPRange iprange.Range
	Iface   string
}

type Speaker interface {
	SetBalancer(ip string, nexthops []corev1.Node) error
	DelBalancer(ip string) error
	Start(stopCh <-chan struct{}) error
	ConfigureWithEIP(config Config, deleted bool) error
}
