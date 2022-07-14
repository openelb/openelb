package service

import (
	"context"

	"github.com/openelb/openelb/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	client client.Client
}

// NewEipService returns a new instance of eipService which implements
// the EipService interface. This is used to register the endpoints to
// the router.
func NewEipService(client client.Client) *eipService {
	return &eipService{
		client: client,
	}
}

// Create creates a new Eip object in the kubernetes cluster.
func (e *eipService) Create(ctx context.Context, eip *v1alpha2.Eip) error {
	return e.client.Create(ctx, eip)
}

// Get returns the Eip object in the kubernetes cluster if found.
func (e *eipService) Get(ctx context.Context, name string) (*v1alpha2.Eip, error) {
	eip := &v1alpha2.Eip{}
	err := e.client.Get(ctx, client.ObjectKey{Name: name}, eip)
	return eip, err
}

// ListEips returns the list of Eip objects in the kubernetes cluster.
func (e *eipService) List(ctx context.Context) (*v1alpha2.EipList, error) {
	eipList := &v1alpha2.EipList{}
	err := e.client.List(ctx, eipList)
	return eipList, err
}

// DeleteEip deletes the Eip object in the kubernetes cluster.
func (e *eipService) Delete(ctx context.Context, eip *v1alpha2.Eip) error {
	return e.client.Delete(ctx, eip)
}
