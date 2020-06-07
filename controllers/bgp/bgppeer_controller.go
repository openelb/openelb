/*
Copyright 2019 The Kubesphere Authors.

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

package bgp

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	bgpserver "github.com/kubesphere/porter/pkg/bgp/serverd"
	"github.com/kubesphere/porter/pkg/constant"
	"github.com/kubesphere/porter/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// BgpPeerReconciler reconciles a BgpPeer object
type BgpPeerReconciler struct {
	client.Client
	Log       logr.Logger
	BgpServer *bgpserver.BgpServer
}

func (r *BgpPeerReconciler) useFinalizerIfNeeded(peer *networkv1alpha1.BgpPeer) (bool, error) {
	if peer.ObjectMeta.DeletionTimestamp.IsZero() {
		if !util.ContainsString(peer.ObjectMeta.Finalizers, constant.FinalizerName) {
			peer.ObjectMeta.Finalizers = append(peer.ObjectMeta.Finalizers, constant.FinalizerName)
			if err := r.Update(context.Background(), peer); err != nil {
				r.Log.Info("Failed to use update to  append finalizer to BgpConf", "service", peer.Name)
				return false, err
			}
			r.Log.Info("Append Finalizer to BgpConf", "ServiceName", peer.Name, "Namespace", peer.Namespace)
		}
	} else {
		// The object is being deleted
		if util.ContainsString(peer.ObjectMeta.Finalizers, constant.FinalizerName) {
			if err := r.BgpServer.DeletePeer(&peer.Spec); err != nil {
				return false, err
			}

			// remove our finalizer from the list and update it.
			peer.ObjectMeta.Finalizers = util.RemoveString(peer.ObjectMeta.Finalizers, constant.FinalizerName)
			if err := r.Update(context.Background(), peer); err != nil {
				if errors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			r.Log.Info("Remove Finalizer before service deleted", "ServiceName", peer.Name, "Namespace", peer.Namespace)
			return true, nil
		}
	}
	return false, nil
}

// +kubebuilder:rbac:groups=network.kubesphere.io,resources=bgppeers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.kubesphere.io,resources=bgppeers/status,verbs=get;update;patch

func (r *BgpPeerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("bgppeer", req.NamespacedName)

	// your logic here
	bgpPeer := &networkv1alpha1.BgpPeer{}
	err := r.Get(context.TODO(), req.NamespacedName, bgpPeer)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	deleted, err := r.useFinalizerIfNeeded(bgpPeer)
	if err != nil {
		return ctrl.Result{}, err
	}

	if deleted {
		return ctrl.Result{}, nil
	}

	err = r.BgpServer.AddOrUpdatePeer(&bgpPeer.Spec)

	return ctrl.Result{RequeueAfter: time.Second * 60}, err
}

func (r *BgpPeerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkv1alpha1.BgpPeer{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return true
			},
		}).Complete(r)
}
