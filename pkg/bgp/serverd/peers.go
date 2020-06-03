package serverd

import (
	"fmt"
	bgpapi "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/nettool"
	api "github.com/osrg/gobgp/api"
	"golang.org/x/net/context"
	"net"
	"reflect"
)

const (
	defaultBgpSendMax     = 8
	defaultBgpRestartTime = 60
)

func (server *BgpServer) EnsureNATChain() error {
	iptable := server.bgpIptable

	chains, err := iptable.ListChains("nat")
	if err != nil {
		return err
	}

	found := false
	for _, chain := range chains {
		if chain == nettool.BgpNatChain {
			found = true
			break
		}
	}
	if !found {
		if err = iptable.NewChain("nat", nettool.BgpNatChain); err != nil {
			return err
		}
	}

	rule := []string{"-j", nettool.BgpNatChain}
	ok, err := iptable.Exists("nat", nettool.BgpNatChain, rule...)
	if err != nil {
		return err
	}
	if !ok {
		err = iptable.Append("nat", "PREROUTING", rule...)
		if err != nil {
			return err
		}
	}

	return iptable.ClearChain("nat", nettool.BgpNatChain)
}

func (server *BgpServer) DeletePeer(neighbor *bgpapi.BgpPeerSpec) error {
	delete := false
	fn := func(peer *api.Peer) {
		if peer.Conf.NeighborAddress == neighbor.Config.NeighborAddress {
			delete = true
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

		if response.Global.As != 0 {
			nettool.DeletePortForwardOfBGP(server.bgpIptable, neighbor.Config.NeighborAddress, "", response.Global.ListenPort)
		}
	}

	return nil
}

func (server *BgpServer) AddOrUpdatePeer(neighbor *bgpapi.BgpPeerSpec) error {
	ip := net.ParseIP(neighbor.Config.NeighborAddress)

	if ip == nil {
		return fmt.Errorf("NeighborAddress is invalid")
	}

	add := true
	foundPeer := &bgpapi.BgpPeerSpec{}
	fn := func(peer *api.Peer) {
		if peer.Conf.NeighborAddress == neighbor.Config.NeighborAddress {
			add = false
			foundPeer = &bgpapi.BgpPeerSpec{
				Config: bgpapi.NeighborConfig{
					PeerAs:          peer.Conf.PeerAs,
					NeighborAddress: peer.Conf.NeighborAddress,
				},
				AddPaths: bgpapi.AddPaths{
					SendMax: uint8(peer.AfiSafis[0].AddPaths.Config.SendMax),
				},
				Transport: bgpapi.Transport{
					PassiveMode: peer.Transport.PassiveMode,
					RemotePort:  uint16(peer.Transport.RemotePort),
				},
				UsingPortForward: neighbor.UsingPortForward,
			}
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

	response, _ := server.bgpServer.GetBgp(context.Background(), nil)
	if add {
		if err := server.bgpServer.AddPeer(context.Background(), &api.AddPeerRequest{
			Peer: peer,
		}); err != nil {
			return err
		}

		if response.Global.As != 0 && neighbor.UsingPortForward {
			nettool.AddPortForwardOfBGP(server.bgpIptable, neighbor.Config.NeighborAddress, "", response.Global.ListenPort)
		}
	} else {
		if neighbor.UsingPortForward {
			nettool.AddPortForwardOfBGP(server.bgpIptable, neighbor.Config.NeighborAddress, "", response.Global.ListenPort)
		} else {
			nettool.DeletePortForwardOfBGP(server.bgpIptable, neighbor.Config.NeighborAddress, "", response.Global.ListenPort)
		}

		if reflect.DeepEqual(foundPeer, neighbor) {
			return nil
		}

		if _, err := server.bgpServer.UpdatePeer(context.Background(), &api.UpdatePeerRequest{
			Peer: peer,
		}); err != nil {
			return err
		}
	}

	return nil
}
