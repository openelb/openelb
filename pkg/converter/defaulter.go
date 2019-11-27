package converter

import (
	bgpapi "github.com/osrg/gobgp/api"
)

func DefaultingPeer(peer *bgpapi.Peer) {
	if peer == nil {
		return
	}
	if len(peer.AfiSafis) == 0 {
		peer.AfiSafis = make([]*bgpapi.AfiSafi, 1)
	}
	peer.AfiSafis[0] = &bgpapi.AfiSafi{
		Config: &bgpapi.AfiSafiConfig{
			Family: &bgpapi.Family{
				Afi:  bgpapi.Family_AFI_IP,
				Safi: bgpapi.Family_SAFI_UNICAST,
			},
		},
		AddPaths: &bgpapi.AddPaths{
			Config: &bgpapi.AddPathsConfig{
				Receive: true,
				SendMax: 8,
			},
		},
	}
}
