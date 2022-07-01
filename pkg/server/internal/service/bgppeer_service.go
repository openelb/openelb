package service

import (
	"context"

	"github.com/openelb/openelb/api/v1alpha2"
)

// BgpPeerService is a interface that is used to manage http requests related to
// BgpPeer.
type BgpPeerService interface {
	// Create creates a new BgpPeer object in the kubernetes cluster.
	Create(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) error
	// Get returns the BgpPeer object in the kubernetes cluster if found.
	Get(ctx context.Context, name string) (*v1alpha2.BgpPeer, error)
	// List returns all the BgpPeer objects in the kubernetes cluster.
	List(ctx context.Context) (*v1alpha2.BgpPeerList, error)
	// Delete deletes the BgpPeer object in the kubernetes cluster.
	Delete(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) error
}

// bgpPeerService is a implementation of the BgpPeerService.
type bgpPeerService struct {
	bgpStore BgpStore
}

// NewBgpPeerService returns a new instance of bgpPeerService which implements
// the BgpPeerService interface. This is used to register the endpoints to
// the router.
func NewBgpPeerService(bgpStore BgpStore) *bgpPeerService {
	return &bgpPeerService{
		bgpStore: bgpStore,
	}
}

// Create creates a new BgpPeer object in the kubernetes cluster.
func (b *bgpPeerService) Create(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) error {
	return b.bgpStore.CreateBgpPeer(ctx, bgpPeer)
}

// Get returns the BgpPeer object in the kubernetes cluster if found.
func (b *bgpPeerService) Get(ctx context.Context, name string) (*v1alpha2.BgpPeer, error) {
	return b.bgpStore.GetBgpPeer(ctx, name)
}

// List returns all the BgpPeer objects in the kubernetes cluster.
func (b *bgpPeerService) List(ctx context.Context) (*v1alpha2.BgpPeerList, error) {
	return b.bgpStore.ListBgpPeer(ctx)
}

// Delete deletes the BgpPeer object in the kubernetes cluster.
func (b *bgpPeerService) Delete(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) error {
	return b.bgpStore.DeleteBgpPeer(ctx, bgpPeer)
}
