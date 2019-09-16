package lb

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ServiceReconciler) reconcile(req types.NamespacedName) error {
	svc := &corev1.Service{}
	err := r.Get(context.TODO(), req, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	r.Log.Info("----------------Begin to reconclie for service")
	deleted, err := r.useFinalizerIfNeeded(svc)

	if err != nil {
		return err
	}
	if deleted {
		return nil
	}

	return r.createLB(svc)
}
