package handler

import (
	"context"

	"github.com/openelb/openelb/api/v1alpha2"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EipHandler is an interface that is used to manage http requests related to
// Eip.
type EipHandler interface {
	// Create creates a new Eip object in the kubernetes cluster.
	Create(ctx context.Context, eip *v1alpha2.Eip) (Create, error)
	// Get returns the Eip object in the kubernetes cluster if found.
	Get(ctx context.Context, name string) (*v1alpha2.Eip, error)
	// List returns the list of Eip objects in the kubernetes cluster.
	List(ctx context.Context) (*v1alpha2.EipList, error)
	// Patch patches the Eip object in the kubernetes cluster.
	Patch(ctx context.Context, name string, patch []byte) (Update, error)
	// Update edits the Eip object in the kubernetes cluster.
	Update(ctx context.Context, name string, newObj *v1alpha2.Eip) (Update, error)
	// Delete deletes the Eip object in the kubernetes cluster.
	Delete(ctx context.Context, name string) (Delete, error)
}

// eipHandler is an implementation of the EipHandler.
type eipHandler struct {
	client client.Client
}

// NewEipHandler returns a new instance of eipHandler which implements
// the EipHandler interface. This is used to register the endpoints to
// the router.
func NewEipHandler(client client.Client) *eipHandler {
	return &eipHandler{
		client: client,
	}
}

// Create creates a new Eip object in the kubernetes cluster.
func (e *eipHandler) Create(ctx context.Context, eip *v1alpha2.Eip) (Create, error) {
	if err := e.client.Create(ctx, eip); err != nil {
		return Create{}, err
	}
	return Create{Created: true}, nil
}

// Get returns the Eip object in the kubernetes cluster if found.
func (e *eipHandler) Get(ctx context.Context, name string) (*v1alpha2.Eip, error) {
	eip := &v1alpha2.Eip{}
	err := e.client.Get(ctx, client.ObjectKey{Name: name}, eip)
	return eip, err
}

// List returns the list of Eip objects in the kubernetes cluster.
func (e *eipHandler) List(ctx context.Context) (*v1alpha2.EipList, error) {
	eipList := &v1alpha2.EipList{}
	err := e.client.List(ctx, eipList)
	return eipList, err
}

// Patch patches the Eip object in the kubernetes cluster.
func (e *eipHandler) Patch(ctx context.Context, name string,
	patch []byte) (Update, error) {
	eip, err := e.Get(ctx, name)
	if err != nil {
		return Update{}, err
	}
	err = e.client.Patch(ctx, eip, client.RawPatch(types.MergePatchType,
		patch))
	if err != nil {
		return Update{}, err
	}
	return Update{Updated: true}, nil
}

// Update edits the Eip object in the kubernetes cluster.
func (e *eipHandler) Update(ctx context.Context, name string,
	newObj *v1alpha2.Eip) (Update, error) {
	eip, err := e.Get(ctx, name)
	if err != nil {
		return Update{}, err
	}
	if newObj.ResourceVersion == "" {
		newObj.ResourceVersion = eip.ResourceVersion
	}
	err = e.client.Update(ctx, newObj)
	if err != nil {
		return Update{}, err
	}
	return Update{Updated: true}, nil
}

// Delete deletes the Eip object in the kubernetes cluster.
func (e *eipHandler) Delete(ctx context.Context, name string) (Delete, error) {
	eip, err := e.Get(ctx, name)
	if err != nil {
		return Delete{}, err
	}
	err = e.client.Delete(ctx, eip)
	if err != nil {
		return Delete{}, err
	}
	return Delete{Deleted: true}, nil
}
