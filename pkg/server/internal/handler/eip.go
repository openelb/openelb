package handler

import (
	"context"

	"github.com/openelb/openelb/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EipHandler is an interface that is used to manage http requests related to
// Eip.
type EipHandler interface {
	// Create creates a new Eip object in the kubernetes cluster.
	Create(ctx context.Context, eip *v1alpha2.Eip) error
	// Get returns the Eip object in the kubernetes cluster if found.
	Get(ctx context.Context, name string) (*v1alpha2.Eip, error)
	// List returns the list of Eip objects in the kubernetes cluster.
	List(ctx context.Context) (*v1alpha2.EipList, error)
	// Delete deletes the Eip object in the kubernetes cluster.
	Delete(ctx context.Context, name string) error
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
func (e *eipHandler) Create(ctx context.Context, eip *v1alpha2.Eip) error {
	return e.client.Create(ctx, eip)
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

// Delete deletes the Eip object in the kubernetes cluster.
func (e *eipHandler) Delete(ctx context.Context, name string) error {
	eip, err := e.Get(ctx, name)
	if err != nil {
		return err
	}
	return e.client.Delete(ctx, eip)
}
