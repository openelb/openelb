package bgp

import (
	"fmt"
	"net"

	bgpapi "github.com/kubesphere/porterlb/api/v1alpha2"
	"github.com/kubesphere/porterlb/pkg/util"
	api "github.com/osrg/gobgp/api"
	"golang.org/x/net/context"
	ctrl "sigs.k8s.io/controller-runtime"
)

func defaultFamily(ip net.IP) *bgpapi.Family {
	family := &bgpapi.Family{
		Afi:  "AFI_IP",
		Safi: "SAFI_UNICAST",
	}
	if ip.To4() == nil {
		family = &bgpapi.Family{
			Afi:  "AFI_IP6",
			Safi: "SAFI_UNICAST",
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
		tmp, err := bgpapi.ConverStatusFromGoBgpPeer(peer)
		if err != nil {
			ctrl.Log.Error(err, "failed to ConverStatusFromGoBgpPeer", "peer", peer)
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
		ctrl.Log.Info("delete useless bgp peer", "peer", del)
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
		ctrl.Log.Error(err, "failed to get bgpconf status")
		return bgpapi.BgpConf{
			Status: bgpapi.BgpConfStatus{
				NodesConfStatus: map[string]bgpapi.NodeConfStatus{
					util.GetNodeName(): bgpapi.NodeConfStatus{
						RouterId: "",
						As:       0,
					},
				},
			},
		}
	}
	return bgpapi.BgpConf{
		Status: bgpapi.BgpConfStatus{
			NodesConfStatus: map[string]bgpapi.NodeConfStatus{
				util.GetNodeName(): bgpapi.NodeConfStatus{
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
	}

	request, e := neighbor.Spec.ConverToGoBgpPeer()
	if e != nil {
		return e
	}

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
