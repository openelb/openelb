/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package eip

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	"github.com/kiali/kiali/log"
	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/constant"
	"github.com/kubesphere/porter/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// +kubebuilder:rbac:groups=network.kubesphere.io,resources=eips,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.kubesphere.io,resources=eips/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// EipReconciler reconciles a Eip object
type EipReconciler struct {
	client.Client
	Log logr.Logger
	record.EventRecorder
}

func (r *EipReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("eip", req.NamespacedName)
	r.Log.Info("----------------Begin to reconclie for eip------------------")
	instance := &networkv1alpha1.Eip{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			r.Log.Info("EIP is deleted safely", "name", instance.GetName(), "namespace", instance.GetNamespace())
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *EipReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkv1alpha1.Eip{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				old := e.ObjectOld.(*networkv1alpha1.Eip)
				new := e.ObjectNew.(*networkv1alpha1.Eip)
				if !e.MetaNew.GetDeletionTimestamp().IsZero() {
					return true
				}
				return old.Status.Occupied != new.Status.Occupied
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
		}).
		Complete(r)
}

func (r *EipReconciler) useFinalizerIfNeeded(eip *networkv1alpha1.Eip) (bool, error) {
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
			r.Log.Info("Append Finalizer to eip", "eipName", eip.Name, "Namespace", eip.Namespace)
			return true, nil
		}
	} else {
		// The object is being deleted
		if util.ContainsString(eip.ObjectMeta.Finalizers, agentFinalizer) {
			r.Log.Info("Begin to remove finalizer")
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
