package bgp

import (
	bgpapi "github.com/kubesphere/porter/api/v1alpha2"
	api "github.com/osrg/gobgp/api"
	"golang.org/x/net/context"
)

func (b *Bgp) HandleBgpGlobalConfig(global *bgpapi.BgpConf, delete bool) error {
	if delete {
		return b.bgpServer.StopBgp(context.Background(), nil)
	}

	request, err := global.Spec.ConverToGoBgpGlabalConf()
	if err != nil {
		return err
	}

	b.bgpServer.StopBgp(context.Background(), nil)
	return b.bgpServer.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: request,
	})
}
