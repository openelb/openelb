package serverd

import (
	"reflect"

	bgpapi "github.com/kubesphere/porter/api/v1alpha1"
	api "github.com/osrg/gobgp/api"
	"golang.org/x/net/context"
)

func (server *BgpServer) HandleBgpGlobalConfig(global *bgpapi.BgpConfSpec, delete bool) error {
	if delete {
		server.bgpServer.StopBgp(context.Background(), nil)
		return nil
	}

	update := false

	response, _ := server.bgpServer.GetBgp(context.Background(), nil)
	if response != nil {
		bgpConf := &bgpapi.BgpConfSpec{
			As:       response.Global.As,
			RouterId: response.Global.RouterId,
			Port:     response.Global.ListenPort,
		}

		if reflect.DeepEqual(bgpConf, global) {
			return nil
		} else {
			update = true
		}
	}

	if update {
		server.bgpServer.StopBgp(context.Background(), nil)
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
