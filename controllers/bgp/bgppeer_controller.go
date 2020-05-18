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
	bgpserver "github.com/kubesphere/porter/pkg/bgp/serverd"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
)

// BgpPeerReconciler reconciles a BgpPeer object
type BgpPeerReconciler struct {
	client.Client
	Log       logr.Logger
	BgpServer *bgpserver.BgpServer
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
			return ctrl.Result{}, r.BgpServer.DeletePeer(&bgpPeer.Spec)
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.BgpServer.AddOrUpdatePeer(&bgpPeer.Spec)
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
			DeleteFunc: func(e event.DeleteEvent) bool {
				return true
			},
		}).Complete(r)
}
