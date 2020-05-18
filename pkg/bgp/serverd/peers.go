package serverd

import (
	"fmt"
	api "github.com/osrg/gobgp/api"
	"golang.org/x/net/context"
	"net"
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
	AddPaths AddPaths `mapstructure:"add-paths" json:"add-paths,omitempty"`

	// original -> bgp:graceful-restart
	// Parameters relating the graceful restart mechanism for BGP.
	GracefulRestart GracefulRestart `mapstructure:"graceful-restart" json:"graceful-restart,omitempty"`

	UsingPortForward bool `json:"using-port-forward,omitempty" mapstructure:"using-port-forward"`
}

// struct for container bgp:config.
// Configuration parameters relating to graceful-restart.
type GracefulRestartConfig struct {
	// original -> bgp:enabled
	// bgp:enabled's original type is boolean.
	// Enable or disable the graceful-restart capability.
	Enabled bool `mapstructure:"enabled" json:"enabled,omitempty"`

	// original -> bgp:restart-time
	// Estimated time (in seconds) for the local BGP speaker to
	// restart a session. This value is advertise in the graceful
	// restart BGP capability.  This is a 12-bit value, referred to
	// as Restart Time in RFC4724.  Per RFC4724, the suggested
	// default value is <= the hold-time value.
	RestartTime uint16 `mapstructure:"restart-time" json:"restart-time,omitempty"`
}

// struct for container bgp:graceful-restart.
// Parameters relating the graceful restart mechanism for BGP.
type GracefulRestart struct {
	// original -> bgp:graceful-restart-config
	// Configuration parameters relating to graceful-restart.
	Config GracefulRestartConfig `mapstructure:"config" json:"config,omitempty"`
}

// struct for container bgp:config.
// Configuration parameters relating to ADD_PATHS.
type AddPathsConfig struct {
	// original -> bgp:send-max
	// The maximum number of paths to advertise to neighbors
	// for a single NLRI.
	SendMax uint8 `mapstructure:"send-max" json:"send-max,omitempty"`
}

// struct for container bgp:state.
// State information associated with ADD_PATHS.
type AddPathsState struct {
	// original -> bgp:send-max
	// The maximum number of paths to advertise to neighbors
	// for a single NLRI.
	SendMax uint8 `mapstructure:"send-max" json:"send-max,omitempty"`
}

// struct for container bgp:add-paths.
// Parameters relating to the advertisement and receipt of
// multiple paths for a single NLRI (add-paths).
type AddPaths struct {
	// original -> bgp:add-paths-config
	// Configuration parameters relating to ADD_PATHS.
	Config AddPathsConfig `mapstructure:"config" json:"config,omitempty"`
}

// struct for container bgp:config.
// Configuration parameters relating to the BGP neighbor or
// group.
type NeighborConfig struct {
	// original -> bgp:peer-as
	// bgp:peer-as's original type is inet:as-number.
	// AS number of the peer.
	PeerAs uint32 `mapstructure:"peer-as" json:"peer-as,omitempty"`

	// original -> bgp:neighbor-address
	// bgp:neighbor-address's original type is inet:ip-address.
	// Address of the BGP peer, either in IPv4 or IPv6.
	NeighborAddress string `mapstructure:"neighbor-address" json:"neighbor-address,omitempty"`
}

func (server *BgpServer) DeletePeer(neighbor *BgpPeerSpec) error {
	if err := server.bgpServer.DeletePeer(context.Background(), &api.DeletePeerRequest{
		Address: neighbor.Config.NeighborAddress,
	}); err != nil {
		return err
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

	family := &api.Family{
		Afi:  api.Family_AFI_IP,
		Safi: api.Family_SAFI_UNICAST,
	}
	if ip.To4() == nil {
		family = &api.Family{
			Afi:  api.Family_AFI_IP6,
			Safi: api.Family_SAFI_UNICAST,
		}
	}

	// neighbor configuration
	sendMax := uint32(neighbor.AddPaths.Config.SendMax)
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
					Family:  family,
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
	}

	if add {
		if err := server.bgpServer.AddPeer(context.Background(), &api.AddPeerRequest{
			Peer: peer,
		}); err != nil {
			return err
		}
	} else {
		if _, err := server.bgpServer.UpdatePeer(context.Background(), &api.UpdatePeerRequest{
			Peer:          peer,
			DoSoftResetIn: true,
		}); err != nil {
			return err
		}
	}

	return nil
}
