package service

import (
	"context"

	"github.com/openelb/openelb/api/v1alpha2"
	"k8s.io/apimachinery/pkg/api/errors"
)

// BgpConfService is an interface that is used to manage http requests related to
// BgpConf.
type BgpConfService interface {
	// Create creates a new BgpConf object in the kubernetes cluster.
	Create(ctx context.Context, bgpConf *v1alpha2.BgpConf) error
	// Get returns the BgpConf object in the kubernetes cluster if found.
	Get(ctx context.Context) (*v1alpha2.BgpConf, error)
	// Delete deletes the BgpConf object in the kubernetes cluster.
	Delete(ctx context.Context, bgpConf *v1alpha2.BgpConf) error
}

// bgpConfService is an implementation of the BgpConfService.
type bgpConfService struct {
	bgpStore BgpStore
}

// NewBgpConfService returns a new instance of bgpConfService which implements
// the BgpConfService interface. This is used to register the endpoints to
// the router.
func NewBgpConfService(bgpStore BgpStore) *bgpConfService {
	return &bgpConfService{
		bgpStore: bgpStore,
	}
}

// Create creates a new BgpConf object in the kubernetes cluster.
func (b *bgpConfService) Create(ctx context.Context,
	bgpConf *v1alpha2.BgpConf) error {
	if bgpConf.Name != "default" {
		return errors.NewBadRequest("BgpConf name must be default")
	}
	if bgpConf.Spec.ListenPort == 0 {
		return errors.NewBadRequest("BgpConf listen port must be set")
	}

	return b.bgpStore.CreateBgpConf(ctx, bgpConf)
}

// Get returns the BgpConf object in the kubernetes cluster if found.
func (b *bgpConfService) Get(ctx context.Context) (*v1alpha2.BgpConf, error) {
	return b.bgpStore.GetBgpConf(ctx, "default")
}

// Delete deletes the BgpConf object in the kubernetes cluster.
func (b *bgpConfService) Delete(ctx context.Context, bgpConf *v1alpha2.BgpConf) error {
	return b.bgpStore.DeleteBgpConf(ctx, bgpConf)
}

// BgpStore is an interface for managing OpenELB Bgp resources.
type BgpStore interface {
	// CreateBgpConf creates a new BgpConf object in the kubernetes cluster.
	CreateBgpConf(ctx context.Context, bgpConf *v1alpha2.BgpConf) error
	// GetBgpConf returns the BgpConf object in the kubernetes cluster if found.
	GetBgpConf(ctx context.Context, name string) (*v1alpha2.BgpConf, error)
	// // UpdateBgpConf updates the BgpConf object in the kubernetes cluster.
	// UpdateBgpConf(ctx context.Context, bgpConf *v1alpha2.BgpConf) error
	// DeleteBgpConf deletes the BgpConf object in the kubernetes cluster.
	DeleteBgpConf(ctx context.Context, bgpConf *v1alpha2.BgpConf) error
	// CreateBgpPeer creates a new BgpPeer object in the kubernetes cluster.
	CreateBgpPeer(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) error
	// GetBgpPeer returns the BgpPeer object in the kubernetes cluster if found.
	GetBgpPeer(ctx context.Context, name string) (*v1alpha2.BgpPeer, error)
	// ListBgpPeers returns a list of BgpPeer objects in the kubernetes cluster.
	ListBgpPeer(ctx context.Context) (*v1alpha2.BgpPeerList, error)
	// DeleteBgpPeer deletes the BgpPeer object in the kubernetes cluster.
	DeleteBgpPeer(ctx context.Context, bgpPeer *v1alpha2.BgpPeer) error
}
