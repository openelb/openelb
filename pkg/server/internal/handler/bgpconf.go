package handler

import (
	"context"
	"encoding/json"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/openelb/openelb/api/v1alpha2"
)

// BgpConfHandler is an interface that is used to manage http requests related to
// BgpConf.
type BgpConfHandler interface {
	// Create creates a new BgpConf object in the kubernetes cluster.
	Create(ctx context.Context, bgpConf *v1alpha2.BgpConf) error
	// Get returns the BgpConf object in the kubernetes cluster if found.
	Get(ctx context.Context) (*v1alpha2.BgpConf, error)
	// Patch patches the BgpConf object in the kubernetes cluster.
	Patch(ctx context.Context, patchObj *v1alpha2.BgpConf) error
	// Delete deletes the BgpConf object in the kubernetes cluster.
	Delete(ctx context.Context) error
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
	bgpConf *v1alpha2.BgpConf) error {
	if bgpConf.Name != "default" {
		return errors.NewBadRequest("BgpConf name must be default")
	}
	if bgpConf.Spec.ListenPort == 0 {
		return errors.NewBadRequest("BgpConf listen port must be set")
	}
	return b.client.Create(ctx, bgpConf)
}

// Get returns the BgpConf object in the kubernetes cluster if found.
func (b *bgpConfHandler) Get(ctx context.Context) (*v1alpha2.BgpConf, error) {
	bgpConf := &v1alpha2.BgpConf{}
	err := b.client.Get(ctx, client.ObjectKey{Name: "default"}, bgpConf)
	return bgpConf, err
}

// Patch patches the BgpConf object in the kubernetes cluster.
func (b *bgpConfHandler) Patch(ctx context.Context,
	patchObj *v1alpha2.BgpConf) error {
	bgpConf, err := b.Get(ctx)
	if err != nil {
		return err
	}
	var patchBytes []byte
	patchBytes, err = json.Marshal(patchObj)
	if err != nil {
		return err
	}
	return b.client.Patch(ctx, bgpConf, client.RawPatch(types.MergePatchType,
		patchBytes))
}

// Delete deletes the BgpConf object in the kubernetes cluster.
func (b *bgpConfHandler) Delete(ctx context.Context) error {
	bgpConf, err := b.Get(ctx)
	if err != nil {
		return err
	}
	return b.client.Delete(ctx, bgpConf)
}
