package serverd

import (
	"fmt"
	"net"

	bgpapi "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/nettool"
	api "github.com/osrg/gobgp/api"
	"golang.org/x/net/context"
)

const (
	defaultBgpSendMax     = 8
	defaultBgpRestartTime = 60
)

func (server *BgpServer) DeletePeer(neighbor *bgpapi.BgpPeerSpec) error {
	delete := false
	fn := func(peer *api.Peer) {
		if peer.Conf.NeighborAddress == neighbor.Config.NeighborAddress {
			delete = false
		}
	}

	server.bgpServer.ListPeer(context.Background(), &api.ListPeerRequest{
		Address: neighbor.Config.NeighborAddress,
	}, fn)

	if delete {
		response, _ := server.bgpServer.GetBgp(context.Background(), nil)

		if err := server.bgpServer.DeletePeer(context.Background(), &api.DeletePeerRequest{
			Address: neighbor.Config.NeighborAddress,
		}); err != nil {
			return err
		}

		return nettool.DeletePortForwardOfBGP(server.bgpIptable, neighbor.Config.NeighborAddress, "", response.Global.ListenPort)
	}

	return nil
}

func (server *BgpServer) AddOrUpdatePeer(neighbor *bgpapi.BgpPeerSpec) error {
	ip := net.ParseIP(neighbor.Config.NeighborAddress)

	if ip == nil {
		return fmt.Errorf("NeighborAddress is invalid")
	}

	add := true
	fn := func(peer *api.Peer) {
		if peer.Conf.NeighborAddress == neighbor.Config.NeighborAddress {
			add = false
		}
	}

	server.bgpServer.ListPeer(context.Background(), &api.ListPeerRequest{
		Address: neighbor.Config.NeighborAddress,
	}, fn)

	// neighbor configuration
	sendMax := uint32(neighbor.AddPaths.SendMax)
	if sendMax <= 0 {
		sendMax = defaultBgpSendMax
	}
	peer := &api.Peer{
		Conf: &api.PeerConf{
			NeighborAddress: neighbor.Config.NeighborAddress,
			PeerAs:          neighbor.Config.PeerAs,
		},
		AfiSafis: []*api.AfiSafi{
			&api.AfiSafi{
				Config: &api.AfiSafiConfig{
					Family:  getFamily(neighbor.Config.NeighborAddress),
					Enabled: true,
				},
				AddPaths: &api.AddPaths{
					Config: &api.AddPathsConfig{
						SendMax: sendMax,
					},
				},
				MpGracefulRestart: &api.MpGracefulRestart{
					Config: &api.MpGracefulRestartConfig{
						Enabled: server.bgpOptions.GracefulRestart,
					},
				},
			},
		},
		GracefulRestart: &api.GracefulRestart{
			Enabled:     server.bgpOptions.GracefulRestart,
			RestartTime: defaultBgpRestartTime,
		},
		Transport: &api.Transport{
			PassiveMode: neighbor.Transport.PassiveMode,
			RemotePort:  uint32(neighbor.Transport.RemotePort),
		},
	}

	if add {
		if err := server.bgpServer.AddPeer(context.Background(), &api.AddPeerRequest{
			Peer: peer,
		}); err != nil {
			return err
		}

		response, _ := server.bgpServer.GetBgp(context.Background(), nil)

		return nettool.AddPortForwardOfBGP(server.bgpIptable, neighbor.Config.NeighborAddress, "", response.Global.ListenPort)
	} else {
		if _, err := server.bgpServer.UpdatePeer(context.Background(), &api.UpdatePeerRequest{
			Peer: peer,
		}); err != nil {
			return err
		}
	}

	return nil
}
