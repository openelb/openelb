package serverd

import (
	"fmt"
	"net"

	"github.com/kubesphere/porter/pkg/nettool"
	api "github.com/osrg/gobgp/api"
	"golang.org/x/net/context"
)

const (
	defaultBgpSendMax     = 8
	defaultBgpRestartTime = 60
)

type BgpPeerSpec struct {
	// original -> bgp:neighbor-address
	// original -> bgp:neighbor-config
	// Configuration parameters relating to the BGP neighbor or
	// group.
	Config NeighborConfig `mapstructure:"config" json:"config,omitempty"`

	// original -> bgp:add-paths
	// Parameters relating to the advertisement and receipt of
	// multiple paths for a single NLRI (add-paths).
	AddPaths AddPaths `mapstructure:"add-paths" json:"addPaths,omitempty"`

	// original -> bgp:transport
	// Transport session parameters for the BGP neighbor or group.
	Transport Transport `mapstructure:"transport" json:"transport,omitempty"`

	UsingPortForward bool `json:"usingPortForward,omitempty" mapstructure:"using-port-forward"`
}

// struct for container bgp:transport.
// Transport session parameters for the BGP neighbor or group.
type Transport struct {
	// original -> bgp:passive-mode
	// bgp:passive-mode's original type is boolean.
	// Wait for peers to issue requests to open a BGP session,
	// rather than initiating sessions from the local router.
	PassiveMode bool `mapstructure:"passive-mode" json:"passiveMode,omitempty"`

	// original -> gobgp:remote-port
	// gobgp:remote-port's original type is inet:port-number.
	RemotePort uint16 `mapstructure:"remote-port" json:"remotePort,omitempty"`
}

// struct for container bgp:add-paths.
// Parameters relating to the advertisement and receipt of
// multiple paths for a single NLRI (add-paths).
type AddPaths struct {
	// original -> bgp:send-max
	// The maximum number of paths to advertise to neighbors
	// for a single NLRI.
	SendMax uint8 `mapstructure:"send-max" json:"sendMax,omitempty"`
}

// struct for container bgp:config.
// Configuration parameters relating to the BGP neighbor or
// group.
type NeighborConfig struct {
	// original -> bgp:peer-as
	// bgp:peer-as's original type is inet:as-number.
	// AS number of the peer.
	PeerAs uint32 `mapstructure:"peer-as" json:"peerAs,omitempty"`

	// original -> bgp:neighbor-address
	// bgp:neighbor-address's original type is inet:ip-address.
	// Address of the BGP peer, either in IPv4 or IPv6.
	NeighborAddress string `mapstructure:"neighbor-address" json:"neighborAddress,omitempty"`
}

func (server *BgpServer) DeletePeer(neighbor *BgpPeerSpec) error {
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

func (server *BgpServer) AddOrUpdatePeer(neighbor *BgpPeerSpec) error {
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
