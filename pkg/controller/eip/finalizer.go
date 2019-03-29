package eip

import (
	"context"
	"os"

	networkv1alpha1 "github.com/kubesphere/porter/pkg/apis/network/v1alpha1"
	"github.com/kubesphere/porter/pkg/controller/constant"
	"github.com/kubesphere/porter/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
)

func (r *ReconcileEIP) useFinalizerIfNeeded(eip *networkv1alpha1.EIP) (bool, error) {
	nodeName := os.Getenv("MY_NODE_NAME")
	agentFinalizer := constant.NodeFinalizerName + "/" + nodeName
	if eip.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object.
		if !util.ContainsString(eip.ObjectMeta.Finalizers, agentFinalizer) {
			eip.ObjectMeta.Finalizers = append(eip.ObjectMeta.Finalizers, agentFinalizer)
			if err := r.Update(context.Background(), eip); err != nil {
				return false, err
			}
			log.Info("Append Finalizer to eip", "eipName", eip.Name, "Namespace", eip.Namespace)
			return true, nil
		}
	} else {
		// The object is being deleted
		if util.ContainsString(eip.ObjectMeta.Finalizers, agentFinalizer) {
			log.Info("Begin to remove finalizer")
			// our finalizer is present, so lets handle our external dependency
			if err := r.DeleteRule(eip); err != nil {
				log.Error(nil, "Failed to delete route", "name", eip.GetName(), "namespace", eip.GetNamespace())
				return true, err
			}
			// remove our finalizer from the list and update it.
			eip.ObjectMeta.Finalizers = util.RemoveString(eip.ObjectMeta.Finalizers, agentFinalizer)
			if err := r.Update(context.Background(), eip); err != nil {
				if errors.IsNotFound(err) {
					return true, nil
				}
				return true, err
			}
			log.Info("Remove Finalizer before eip deleted", "eipName", eip.Name, "Namespace", eip.Namespace)
			return true, nil
		}
	}
	return false, nil
}
