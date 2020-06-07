package serverd

import (
	"reflect"

	bgpapi "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/nettool"
	api "github.com/osrg/gobgp/api"
	"golang.org/x/net/context"
)

func (server *BgpServer) HandleBgpGlobalConfig(global *bgpapi.BgpConfSpec, delete bool) error {
	update := false

	response, _ := server.bgpServer.GetBgp(context.Background(), nil)
	if response.Global.As != 0 {
		bgpConf := &bgpapi.BgpConfSpec{
			As:       response.Global.As,
			RouterId: response.Global.RouterId,
			Port:     response.Global.ListenPort,
		}

		if bgpConf.As != global.As || (reflect.DeepEqual(bgpConf, global) && !delete) {
			return nil
		} else {
			update = true
		}
	}

	if delete || update {
		fn := func(peer *api.Peer) {
			nettool.DeletePortForwardOfBGP(server.bgpIptable, peer.Conf.NeighborAddress, "", response.Global.ListenPort)
		}
		server.bgpServer.ListPeer(context.Background(), &api.ListPeerRequest{}, fn)
		server.bgpServer.StopBgp(context.Background(), nil)
		if delete {
			return nil
		}
	}

	if err := server.bgpServer.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: &api.Global{
			As:         global.As,
			RouterId:   global.RouterId,
			ListenPort: global.Port,
		},
	}); err != nil {
		return err
	}

	return nil
}
