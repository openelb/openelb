package handler

import (
	"context"

	"github.com/openelb/openelb/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BgpPeerHandler is an interface that is used to manage http requests related to
// BgpPeer.
type BgpPeerHandler interface {
	// Create creates a new BgpPeer object in the kubernetes cluster.
	Create(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) error
	// Get returns the BgpPeer object in the kubernetes cluster if found.
	Get(ctx context.Context, name string) (*v1alpha2.BgpPeer, error)
	// List returns all the BgpPeer objects in the kubernetes cluster.
	List(ctx context.Context) (*v1alpha2.BgpPeerList, error)
	// Delete deletes the BgpPeer object in the kubernetes cluster.
	Delete(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) error
}

// bgpPeerHandler is an implementation of the BgpPeerHandler.
type bgpPeerHandler struct {
	client client.Client
}

// NewBgpPeerHandler returns a new instance of bgpPeerHandler which implements
// the BgpPeerHandler interface. This is used to register the endpoints to
// the router.
func NewBgpPeerHandler(client client.Client) *bgpPeerHandler {
	return &bgpPeerHandler{
		client: client,
	}
}

// Create creates a new BgpPeer object in the kubernetes cluster.
func (b *bgpPeerHandler) Create(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) error {
	return b.client.Create(ctx, bgpPeer)
}

// Get returns the BgpPeer object in the kubernetes cluster if found.
func (b *bgpPeerHandler) Get(ctx context.Context, name string) (*v1alpha2.BgpPeer, error) {
	bgpPeer := &v1alpha2.BgpPeer{}
	err := b.client.Get(ctx, client.ObjectKey{Name: name}, bgpPeer)
	return bgpPeer, err
}

// List returns all the BgpPeer objects in the kubernetes cluster.
func (b *bgpPeerHandler) List(ctx context.Context) (*v1alpha2.BgpPeerList, error) {
	bgpPeers := &v1alpha2.BgpPeerList{}
	err := b.client.List(ctx, bgpPeers)
	return bgpPeers, err
}

// Delete deletes the BgpPeer object in the kubernetes cluster.
func (b *bgpPeerHandler) Delete(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) error {
	return b.client.Delete(ctx, bgpPeer)
}
