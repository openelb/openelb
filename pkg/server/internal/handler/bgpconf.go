package handler

import (
	"context"

	"github.com/openelb/openelb/api/v1alpha2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BgpConfHandler is an interface that is used to manage http requests related to
// BgpConf.
type BgpConfHandler interface {
	// Create creates a new BgpConf object in the kubernetes cluster.
	Create(ctx context.Context, bgpConf *v1alpha2.BgpConf) (Create, error)
	// Get returns the BgpConf object in the kubernetes cluster if found.
	Get(ctx context.Context) (*v1alpha2.BgpConf, error)
	// Patch patches the BgpConf object in the kubernetes cluster.
	Patch(ctx context.Context, patch []byte) (Update, error)
	// Update edits the BgpConf object in the kubernetes cluster.
	Update(ctx context.Context, newObj *v1alpha2.BgpConf) (Update, error)
	// Delete deletes the BgpConf object in the kubernetes cluster.
	Delete(ctx context.Context) (Delete, error)
}

// bgpConfHandler is an implementation of the BgpConfHandler.
type bgpConfHandler struct {
	client client.Client
}

// NewBgpConfHandler returns a new instance of bgpConfHandler which implements
// the BgpConfHandler interface. This is used to register the endpoints to
// the router.
func NewBgpConfHandler(client client.Client) *bgpConfHandler {
	return &bgpConfHandler{
		client: client,
	}
}

// Create creates a new BgpConf object in the kubernetes cluster.
func (b *bgpConfHandler) Create(ctx context.Context,
	bgpConf *v1alpha2.BgpConf) (Create, error) {
	if bgpConf.Name != "default" {
		return Create{}, errors.NewBadRequest("BgpConf name must be default")
	}
	if bgpConf.Spec.ListenPort == 0 {
		return Create{}, errors.NewBadRequest("BgpConf listen port must be set")
	}
	if err := b.client.Create(ctx, bgpConf); err != nil {
		return Create{}, err
	}
	return Create{Created: true}, nil
}

// Get returns the BgpConf object in the kubernetes cluster if found.
func (b *bgpConfHandler) Get(ctx context.Context) (*v1alpha2.BgpConf, error) {
	bgpConf := &v1alpha2.BgpConf{}
	err := b.client.Get(ctx, client.ObjectKey{Name: "default"}, bgpConf)
	return bgpConf, err
}

// Patch patches the BgpConf object in the kubernetes cluster.
func (b *bgpConfHandler) Patch(ctx context.Context, patch []byte) (Update, error) {
	bgpConf, err := b.Get(ctx)
	if err != nil {
		return Update{}, err
	}
	err = b.client.Patch(ctx, bgpConf, client.RawPatch(types.MergePatchType,
		patch))
	if err != nil {
		return Update{}, err
	}
	return Update{Updated: true}, nil
}

// Update edits the BgpConf object in the kubernetes cluster.
func (b *bgpConfHandler) Update(ctx context.Context,
	newObj *v1alpha2.BgpConf) (Update, error) {
	bgpConf, err := b.Get(ctx)
	if err != nil {
		return Update{}, err
	}
	if newObj.ResourceVersion == "" {
		newObj.ResourceVersion = bgpConf.ResourceVersion
	}
	err = b.client.Update(ctx, newObj)
	if err != nil {
		return Update{}, err
	}
	return Update{Updated: true}, nil
}

// Delete deletes the BgpConf object in the kubernetes cluster.
func (b *bgpConfHandler) Delete(ctx context.Context) (Delete, error) {
	bgpConf, err := b.Get(ctx)
	if err != nil {
		return Delete{}, err
	}
	err = b.client.Delete(ctx, bgpConf)
	if err != nil {
		return Delete{}, err
	}
	return Delete{Deleted: true}, nil
}
