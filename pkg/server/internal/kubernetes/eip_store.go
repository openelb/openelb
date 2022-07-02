package kubernetes

import (
	"context"

	"github.com/openelb/openelb/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// eipStore is a kubernetes based implementation of the EipStore interface.
type eipStore struct {
	client client.Client
}

// NewEipStore returns a kubernetes based implementation of the EipStore interface.
func NewEipStore(client client.Client) *eipStore {
	return &eipStore{
		client,
	}
}

// CreateEip creates a new Eip object in the kubernetes cluster.
func (e *eipStore) CreateEip(ctx context.Context, eip *v1alpha2.Eip) error {
	err := e.client.Create(ctx, eip)
	return err
}

// GetEip returns the Eip object in the kubernetes cluster if found.
func (e *eipStore) GetEip(ctx context.Context, name string) (*v1alpha2.Eip, error) {
	eip := &v1alpha2.Eip{}
	err := e.client.Get(ctx, client.ObjectKey{Name: name}, eip)
	return eip, err
}

// ListEips returns the list of Eip objects in the kubernetes cluster.
func (e *eipStore) ListEip(ctx context.Context) (*v1alpha2.EipList, error) {
	eipList := &v1alpha2.EipList{}
	err := e.client.List(ctx, eipList)
	return eipList, err
}

// DeleteEip deletes the Eip object in the kubernetes cluster.
func (e *eipStore) DeleteEip(ctx context.Context, eip *v1alpha2.Eip) error {
	err := e.client.Delete(ctx, eip)
	return err
}
