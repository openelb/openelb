package bgpwrapper

import (
	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	bgpapi "github.com/osrg/gobgp/api"
)

type Interface interface {
	GetPeer(p *networkv1alpha1.BgpPeer) (*bgpapi.Peer, error)
	ListPeers() ([]*bgpapi.Peer, error)
	AddPeer(p *networkv1alpha1.BgpPeer) error
	DeletePeer(p *networkv1alpha1.BgpPeer) error
	UpdatePeer(p *networkv1alpha1.BgpPeer) error
	NeedUpdate(p *networkv1alpha1.BgpPeer) (bool, error)
}
