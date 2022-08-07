package bgp

import (
	"context"

	"github.com/openelb/openelb/pkg/metrics"
	"github.com/openelb/openelb/pkg/util"
	api "github.com/osrg/gobgp/api"
)

func (b *Bgp) getPeers() []*api.Peer {
	peerList := []*api.Peer{}
	fn := func(p *api.Peer) {
		peerList = append(peerList, p)
	}
	err := b.bgpServer.ListPeer(context.Background(), &api.ListPeerRequest{}, fn)
	if err != nil {
		return nil
	}
	return peerList
}

func (b *Bgp) filterPeers(peers []*api.Peer) (map[string]api.Peer, map[string]api.Peer) {
	var (
		staleNodeMap map[string]api.Peer
		nodeMap      map[string]api.Peer
	)
	fn := func(peer *api.Peer) {
		var found *api.Peer
		for _, peer := range peers {
			if peer.Conf.NeighborAddress == peer.State.NeighborAddress {
				found = peer
				break
			}
		}
		if found == nil {
			staleNodeMap[util.GetNodeName()] = *peer
		} else if found.Conf.NeighborAddress == found.State.NeighborAddress {
			nodeMap[util.GetNodeName()] = *found
		}
	}
	b.bgpServer.ListPeer(context.Background(), &api.ListPeerRequest{
		Address: "",
	}, fn)
	return nodeMap, staleNodeMap
}

func (b *Bgp) UpdatePeerMetrics() {
	nodeMap, staleNodeMap := b.filterPeers(b.getPeers())
	updatePeerMetrics(nodeMap)
	deletePeerMetrics(staleNodeMap)
}

func updatePeerMetrics(nodes map[string]api.Peer) {
	for node, peer := range nodes {
		peerIP := peer.Conf.NeighborAddress
		if node != util.GetNodeName() {
			continue
		}
		state := float64(peer.State.SessionState)
		updateCount := float64(peer.State.Messages.Received.Update)
		metrics.UpdateBGPSessionMetrics(peerIP, node, state, updateCount)
	}
}

func deletePeerMetrics(staleNodes map[string]api.Peer) {
	for node, peer := range staleNodes {
		peerIP := peer.Conf.NeighborAddress
		metrics.DeleteBGPPeerMetrics(peerIP, node)
	}
}
