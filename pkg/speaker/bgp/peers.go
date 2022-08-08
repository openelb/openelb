package bgp

import (
	api "github.com/osrg/gobgp/api"
	"golang.org/x/net/context"
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
