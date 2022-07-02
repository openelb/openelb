package service

import (
	"context"

	"github.com/openelb/openelb/api/v1alpha2"
)

// EipService is an interface that is used to manage http requests related to
// Eip.
type EipService interface {
	// Create creates a new Eip object in the kubernetes cluster.
	Create(ctx context.Context, eip *v1alpha2.Eip) error
	// Get returns the Eip object in the kubernetes cluster if found.
	Get(ctx context.Context, name string) (*v1alpha2.Eip, error)
	// ListEips returns the list of Eip objects in the kubernetes cluster.
	List(ctx context.Context) (*v1alpha2.EipList, error)
	// DeleteEip deletes the Eip object in the kubernetes cluster.
	Delete(ctx context.Context, eip *v1alpha2.Eip) error
}

// eipService is an implementation of the EipService.
type eipService struct {
	eipStore EipStore
}

// NewEipService returns a new instance of eipService which implements
// the EipService interface. This is used to register the endpoints to
// the router.
func NewEipService(eipStore EipStore) *eipService {
	return &eipService{
		eipStore: eipStore,
	}
}

// Create creates a new Eip object in the kubernetes cluster.
func (e *eipService) Create(ctx context.Context, eip *v1alpha2.Eip) error {
	return e.eipStore.CreateEip(ctx, eip)
}

// Get returns the Eip object in the kubernetes cluster if found.
func (e *eipService) Get(ctx context.Context, name string) (*v1alpha2.Eip, error) {
	return e.eipStore.GetEip(ctx, name)
}

// ListEips returns the list of Eip objects in the kubernetes cluster.
func (e *eipService) List(ctx context.Context) (*v1alpha2.EipList, error) {
	return e.eipStore.ListEip(ctx)
}

// DeleteEip deletes the Eip object in the kubernetes cluster.
func (e *eipService) Delete(ctx context.Context, eip *v1alpha2.Eip) error {
	return e.eipStore.DeleteEip(ctx, eip)
}

// EipStore is an interface for managing OpenELB EIP rerources.
type EipStore interface {
	// CreateEip creates a new Eip object in the kubernetes cluster.
	CreateEip(ctx context.Context, eip *v1alpha2.Eip) error
	// GetEip returns the Eip object in the kubernetes cluster if found.
	GetEip(ctx context.Context, name string) (*v1alpha2.Eip, error)
	// ListEips returns the list of Eip objects in the kubernetes cluster.
	ListEip(ctx context.Context) (*v1alpha2.EipList, error)
	// DeleteEip deletes the Eip object in the kubernetes cluster.
	DeleteEip(ctx context.Context, eip *v1alpha2.Eip) error
}
