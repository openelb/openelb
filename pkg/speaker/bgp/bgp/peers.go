package bgp

import (
	"fmt"
	"net"
	"strconv"

	bgpapi "github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/metrics"
	"github.com/openelb/openelb/pkg/util"
	api "github.com/osrg/gobgp/api"
	"golang.org/x/net/context"
	"k8s.io/klog/v2"
)

func defaultFamily(ip net.IP) *bgpapi.Family {
	family := &bgpapi.Family{
		Afi:  api.Family_AFI_IP.String(),
		Safi: api.Family_SAFI_UNICAST.String(),
	}
	if ip.To4() == nil {
		family = &bgpapi.Family{
			Afi:  api.Family_AFI_IP6.String(),
			Safi: api.Family_SAFI_UNICAST.String(),
		}
	}

	return family
}

func (b *Bgp) HandleBgpPeerStatus(bgpPeers []bgpapi.BgpPeer) []*bgpapi.BgpPeer {
	var (
		result []*bgpapi.BgpPeer
		dels   []*api.Peer
	)

	fn := func(peer *api.Peer) {
		tmp, err := bgpapi.GetStatusFromGoBgpPeer(peer)
		if err != nil {
			klog.Errorf("failed to GetStatusFromGoBgpPeer: %v", err)
			return
		}

		var found *bgpapi.BgpPeer

		for _, bgpPeer := range bgpPeers {
			if bgpPeer.Spec.Conf.NeighborAddress == tmp.PeerState.NeighborAddress {
				found = &bgpPeer
				break
			}
		}

		if found == nil {
			dels = append(dels, peer)
		} else {
			clone := found.DeepCopy()
			if clone.Status.NodesPeerStatus == nil {
				clone.Status.NodesPeerStatus = make(map[string]bgpapi.NodePeerStatus)
			}

			if clone.Spec.Conf.NeighborAddress == tmp.PeerState.NeighborAddress {
				clone.Status.NodesPeerStatus[util.GetNodeName()] = tmp
			}

			result = append(result, clone)
		}
	}
	b.bgpServer.ListPeer(context.Background(), &api.ListPeerRequest{
		Address: "",
	}, fn)

	for _, del := range dels {
		klog.Infof("delete useless bgp peer: %s", del.Conf.NeighborAddress)
		b.bgpServer.DeletePeer(context.Background(), &api.DeletePeerRequest{
			Address:   del.Conf.NeighborAddress,
			Interface: del.Conf.NeighborInterface,
		})
	}

	return result
}

func (b *Bgp) GetBgpConfStatus() bgpapi.BgpConf {
	result, err := b.bgpServer.GetBgp(context.Background(), nil)
	if err != nil {
		klog.Errorf("failed to get bgpconf status: %v", err)
		return bgpapi.BgpConf{
			Status: bgpapi.BgpConfStatus{
				NodesConfStatus: map[string]bgpapi.NodeConfStatus{
					util.GetNodeName(): {RouterId: "", As: 0},
				},
			},
		}
	}
	return bgpapi.BgpConf{
		Status: bgpapi.BgpConfStatus{
			NodesConfStatus: map[string]bgpapi.NodeConfStatus{
				util.GetNodeName(): {
					RouterId: result.Global.RouterId,
					As:       result.Global.As,
				},
			},
		},
	}
}

func (b *Bgp) HandleBgpPeer(neighbor *bgpapi.BgpPeer, delete bool) error {
	// set default afisafi
	if len(neighbor.Spec.AfiSafis) == 0 {
		ip := net.ParseIP(neighbor.Spec.Conf.NeighborAddress)
		if ip == nil {
			return fmt.Errorf("field Spec.Conf.NeighborAddress invalid")
		}
		neighbor.Spec.AfiSafis = append(neighbor.Spec.AfiSafis, &bgpapi.AfiSafi{
			Config: &bgpapi.AfiSafiConfig{
				Family:  defaultFamily(ip),
				Enabled: true,
			},
			AddPaths: &bgpapi.AddPaths{
				Config: &bgpapi.AddPathsConfig{
					SendMax: 10,
				},
			},
		})
	} else {
		for i := 0; i < len(neighbor.Spec.AfiSafis); i++ {
			if neighbor.Spec.AfiSafis[i].Config == nil {
				ip := net.ParseIP(neighbor.Spec.Conf.NeighborAddress)
				if ip == nil {
					return fmt.Errorf("field Spec.Conf.NeighborAddress invalid")
				}
				neighbor.Spec.AfiSafis[i].Config = &bgpapi.AfiSafiConfig{
					Family:  defaultFamily(ip),
					Enabled: true,
				}
			}
		}
	}

	request, e := neighbor.Spec.ToGoBgpPeer()
	if e != nil {
		return e
	}

	b.UpdatePeerMetrics(neighbor, delete)
	if delete {
		b.bgpServer.DeletePeer(context.Background(), &api.DeletePeerRequest{
			Address:   request.Conf.NeighborAddress,
			Interface: request.Conf.NeighborInterface,
		})
	} else {
		_, e = b.bgpServer.UpdatePeer(context.Background(), &api.UpdatePeerRequest{
			Peer: request,
		})
		if e != nil {
			return b.bgpServer.AddPeer(context.Background(), &api.AddPeerRequest{
				Peer: request,
			})
		}
	}

	return nil
}

func (b *Bgp) UpdatePeerMetrics(peer *bgpapi.BgpPeer, delete bool) {
	status := peer.Status
	for node, peerStatus := range status.NodesPeerStatus {
		var state float64 = 0
		peerIP := peer.Spec.Conf.NeighborAddress
		if node != util.GetNodeName() {
			continue
		}

		if delete {
			metrics.DeleteBGPPeerMetrics(peerIP, node)
			continue
		}

		stateStr := peerStatus.PeerState.SessionState
		switch stateStr {
		case "IDLE":
			state = 0
		case "CONNECT":
			state = 1
		case "ACTIVE":
			state = 2
		case "OPENSENT":
			state = 3
		case "OPENCONFIRM":
			state = 4
		case "ESTABLISHED":
			state = 5
		}

		updateCount, _ := strconv.ParseFloat(peerStatus.PeerState.Messages.Received.Update, 64)
		metrics.UpdateBGPSessionMetrics(peerIP, node, state, updateCount)
	}
}
