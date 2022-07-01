package kubernetes

import (
	"context"

	"github.com/openelb/openelb/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// bgpStore is a kubernetes based implementation of the BgpStore interface.
type bgpStore struct {
	client client.Client
}

// NewBgpStore returns a kubernetes based implementation of the BgpStore.
// interface.
func NewBgpStore(client client.Client) *bgpStore {
	return &bgpStore{
		client,
	}
}

// CreateBgpConf creates a new BgpConf object in the kubernetes cluster.
func (b *bgpStore) CreateBgpConf(ctx context.Context, bgpConf *v1alpha2.BgpConf) error {
	err := b.client.Create(ctx, bgpConf)
	return err
}

// GetBgpConf returns the BgpConf object in the kubernetes cluster if found.
func (b *bgpStore) GetBgpConf(ctx context.Context, name string) (*v1alpha2.BgpConf, error) {
	bgpConf := &v1alpha2.BgpConf{}
	err := b.client.Get(ctx, client.ObjectKey{Name: name}, bgpConf)
	return bgpConf, err
}

// // UpdateBgpConf updates the BgpConf object in the kubernetes cluster.
// func (b *bgpStore) UpdateBgpConf(ctx context.Context, bgpConf *v1alpha2.BgpConf) error {
// 	err := b.client.Update(ctx, bgpConf)
// 	return err
// }

// DeleteBgpConf deletes the BgpConf object in the kubernetes cluster.
func (b *bgpStore) DeleteBgpConf(ctx context.Context, bgpConf *v1alpha2.BgpConf) error {
	err := b.client.Delete(ctx, bgpConf)
	return err
}

// CreateBgpPeer creates a new BgpPeer object in the kubernetes cluster.
func (b *bgpStore) CreateBgpPeer(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) error {
	err := b.client.Create(ctx, bgpPeer)
	return err
}

// GetBgpPeer returns the BgpPeer object in the kubernetes cluster if found.
func (b *bgpStore) GetBgpPeer(ctx context.Context, name string) (*v1alpha2.BgpPeer, error) {
	bgpPeer := &v1alpha2.BgpPeer{}
	err := b.client.Get(ctx, client.ObjectKey{Name: name}, bgpPeer)
	return bgpPeer, err
}

// ListBgpPeers returns a list of BgpPeer objects in the kubernetes cluster.
func (b *bgpStore) ListBgpPeer(ctx context.Context) (*v1alpha2.BgpPeerList, error) {
	bgpPeers := &v1alpha2.BgpPeerList{}
	err := b.client.List(ctx, bgpPeers)
	return bgpPeers, err
}

// DeleteBgpPeer deletes the BgpPeer object in the kubernetes cluster.
func (b *bgpStore) DeleteBgpPeer(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) error {
	err := b.client.Delete(ctx, bgpPeer)
	return err
}
