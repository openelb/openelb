package bgpwrapper

import (
	"context"
	"reflect"

	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/converter"
	"github.com/kubesphere/porter/pkg/errors"
	bgpapi "github.com/osrg/gobgp/api"
	"github.com/osrg/gobgp/pkg/server"
)

// NewBGP create a wrapper of gobgp api
func NewBGP(s *server.BgpServer) Interface {
	return &impl{server: s}
}

type impl struct {
	server *server.BgpServer
}

func (i *impl) GetPeer(peer *networkv1alpha1.BgpPeer) (output *bgpapi.Peer, err error) {
	request := &bgpapi.ListPeerRequest{
		Address: peer.Spec.Conf.NeighborAddress,
	}
	found := false
	callback := func(p *bgpapi.Peer) {
		found = true
		output = p
	}
	err = i.server.ListPeer(context.TODO(), request, callback)
	if err != nil {
		return
	}
	if found {
		return
	}
	return nil, errors.NewResourceNotFoundError("peer", peer.Spec.Conf.NeighborAddress)
}

func (i *impl) ListPeers() ([]*bgpapi.Peer, error) {
	output := make([]*bgpapi.Peer, 0)
	request := &bgpapi.ListPeerRequest{}
	callback := func(p *bgpapi.Peer) {
		temp := *p
		output = append(output, &temp)
	}
	err := i.server.ListPeer(context.TODO(), request, callback)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (i *impl) AddPeer(p *networkv1alpha1.BgpPeer) error {
	peer := converter.ConvertPeerToGoBGP(p)
	converter.DefaultingPeer(peer)
	request := &bgpapi.AddPeerRequest{
		Peer: peer,
	}
	err := i.server.AddPeer(context.Background(), request)
	if err != nil {
		return err
	}
	return nil
}

// Delete peer will fail if the address is not existed
func (i *impl) DeletePeer(peer *networkv1alpha1.BgpPeer) error {
	request := &bgpapi.DeletePeerRequest{
		Address: peer.Spec.Conf.NeighborAddress,
	}
	return i.server.DeletePeer(context.Background(), request)
}

func (i *impl) UpdatePeer(p *networkv1alpha1.BgpPeer) error {
	peer := converter.ConvertPeerToGoBGP(p)
	converter.DefaultingPeer(peer)
	r := &bgpapi.UpdatePeerRequest{
		Peer: peer,
	}
	_, err := i.server.UpdatePeer(context.Background(), r)
	if err != nil {
		return err
	}
	return nil
}

func (i *impl) NeedUpdate(p *networkv1alpha1.BgpPeer) (bool, error) {
	gbPeer, err := i.GetPeer(p)
	if err != nil {
		if errors.IsResourceNotFound(err) {
			return false, nil
		}
		return false, err
	}
	toCompare := converter.ConvertPeerFromGoBGP(gbPeer)
	return reflect.DeepEqual(toCompare.Spec, p.Spec), nil
}
