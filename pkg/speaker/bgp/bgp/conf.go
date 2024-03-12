package bgp

import (
	bgpapi "github.com/openelb/openelb/api/v1alpha2"
	api "github.com/osrg/gobgp/api"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
)

func (b *Bgp) HandleBgpGlobalConfig(global *bgpapi.BgpConf, rack string, delete bool, cm *corev1.ConfigMap) error {
	b.rack = rack

	if delete {
		return b.bgpServer.StopBgp(context.Background(), nil)
	}

	request, err := global.Spec.ToGoBgpGlobalConf()
	if err != nil {
		return err
	}

	b.bgpServer.StopBgp(context.Background(), nil)
	err = b.bgpServer.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: request,
	})
	if err != nil {
		return err
	}
	err = b.updatePolicy(cm)
	if err != nil {
		b.log.Error(err, "failed to update bgp policy")
	}
	return nil
}
