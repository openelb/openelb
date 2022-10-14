package handler

import (
	"context"

	"github.com/openelb/openelb/api/v1alpha2"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BgpPeerHandler is an interface that is used to manage http requests related to
// BgpPeer.
type BgpPeerHandler interface {
	// Create creates a new BgpPeer object in the kubernetes cluster.
	Create(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) (Create, error)
	// Get returns the BgpPeer object in the kubernetes cluster if found.
	Get(ctx context.Context, name string) (*v1alpha2.BgpPeer, error)
	// List returns all the BgpPeer objects in the kubernetes cluster.
	List(ctx context.Context) (*v1alpha2.BgpPeerList, error)
	// Patch patches the BgpPeer object in the kubernetes cluster.
	Patch(ctx context.Context, name string, patch []byte) (Update, error)
	// Update edits the BgpPeer object in the kubernetes cluster.
	Update(ctx context.Context, name string, newObj *v1alpha2.BgpPeer) (Update, error)
	// Delete deletes the BgpPeer object in the kubernetes cluster.
	Delete(ctx context.Context, name string) (Delete, error)
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
func (b *bgpPeerHandler) Create(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) (Create, error) {
	if err := b.client.Create(ctx, bgpPeer); err != nil {
		return Create{}, err
	}
	return Create{Created: true}, nil
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

// Patch patches the BgpPeer object in the kubernetes cluster.
func (b *bgpPeerHandler) Patch(ctx context.Context, name string,
	patch []byte) (Update, error) {
	bgpPeer, err := b.Get(ctx, name)
	if err != nil {
		return Update{}, err
	}
	err = b.client.Patch(ctx, bgpPeer, client.RawPatch(types.MergePatchType,
		patch))
	if err != nil {
		return Update{}, err
	}
	return Update{Updated: true}, nil
}

// Update edits the BgpPeer object in the kubernetes cluster.
func (b *bgpPeerHandler) Update(ctx context.Context, name string,
	newObj *v1alpha2.BgpPeer) (Update, error) {
	bgpPeer, err := b.Get(ctx, name)
	if err != nil {
		return Update{}, err
	}
	if newObj.ResourceVersion == "" {
		newObj.ResourceVersion = bgpPeer.ResourceVersion
	}
	err = b.client.Update(ctx, newObj)
	if err != nil {
		return Update{}, err
	}
	return Update{Updated: true}, nil
}

// Delete deletes the BgpPeer object in the kubernetes cluster.
func (b *bgpPeerHandler) Delete(ctx context.Context, name string) (Delete, error) {
	bgpPeer, err := b.Get(ctx, name)
	if err != nil {
		return Delete{}, err
	}
	err = b.client.Delete(ctx, bgpPeer)
	if err != nil {
		return Delete{}, err
	}
	return Delete{Deleted: true}, nil
}
