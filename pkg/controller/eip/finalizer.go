package eip

import (
	"context"

	networkv1alpha1 "github.com/kubesphere/porter/pkg/apis/network/v1alpha1"
	"github.com/kubesphere/porter/pkg/controller/constant"
	"github.com/kubesphere/porter/pkg/util"
)

func (r *ReconcileEIP) useFinalizerIfNeeded(eip *networkv1alpha1.EIP) (bool, error) {
	if eip.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object.
		if !util.ContainsString(eip.ObjectMeta.Finalizers, constant.FinalizerName) {
			eip.ObjectMeta.Finalizers = append(eip.ObjectMeta.Finalizers, constant.FinalizerName)
			if err := r.Update(context.Background(), eip); err != nil {
				return false, err
			}
			log.Info("Append Finalizer to eip", "eipName", eip.Name, "Namespace", eip.Namespace)
			return true, nil
		}
	} else {
		// The object is being deleted
		if util.ContainsString(eip.ObjectMeta.Finalizers, constant.FinalizerName) {
			// our finalizer is present, so lets handle our external dependency
			if err := r.DelRule(eip); err != nil {
				log.Error(nil, "Failed to delete route", "name", eip.GetName(), "namespace", eip.GetNamespace())
				return true, err
			}
			// remove our finalizer from the list and update it.
			eip.ObjectMeta.Finalizers = util.RemoveString(eip.ObjectMeta.Finalizers, constant.FinalizerName)
			if err := r.Update(context.Background(), eip); err != nil {
				return true, err
			}
			log.Info("Remove Finalizer before eip deleted", "eipName", eip.Name, "Namespace", eip.Namespace)
			return true, nil
		}
	}
	return false, nil
}
