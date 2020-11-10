/*
Copyright 2020 The Kubesphere Authors.

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
	"reflect"
	"time"

	"github.com/kubesphere/porter/api/v1alpha2"
	"github.com/kubesphere/porter/pkg/constant"
	"github.com/kubesphere/porter/pkg/speaker/bgp"
	"github.com/kubesphere/porter/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// BgpPeerReconciler reconciles a BgpPeer object
type BgpPeerReconciler struct {
	client.Client
	BgpServer *bgp.Bgp
	record.EventRecorder
}

// +kubebuilder:rbac:groups=network.kubesphere.io,resources=bgppeers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.kubesphere.io,resources=bgppeers/status,verbs=get;update;patch

func (r BgpPeerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.Log.WithValues("request", req.NamespacedName)

	matchNode := true

	bgpPeer := &v1alpha2.BgpPeer{}
	err := r.Get(context.TODO(), req.NamespacedName, bgpPeer)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//filter peer with nodeSelector
	if bgpPeer.Spec.NodeSelector != nil {
		node := &corev1.Node{}
		err = r.Get(context.Background(), types.NamespacedName{Name: util.GetNodeName()}, node)
		if err != nil {
			return ctrl.Result{}, err
		}
		lbls := labels.Set(node.GetLabels())
		peerSelector, err := metav1.LabelSelectorAsSelector(bgpPeer.Spec.NodeSelector)
		if err != nil {
			return ctrl.Result{}, err
		}
		if !peerSelector.Matches(lbls) {
			matchNode = false
		}
	}

	clone := bgpPeer.DeepCopy()

	if util.IsDeletionCandidate(clone, constant.FinalizerName) {
		err := r.BgpServer.HandleBgpPeer(clone, true)
		if err != nil {
			log.Error(err, "cannot delete bgp peer, maybe need to delete manually")
		}

		controllerutil.RemoveFinalizer(clone, constant.FinalizerName)
		err = r.Update(context.Background(), clone)
		return ctrl.Result{}, err
	}

	if util.NeedToAddFinalizer(clone, constant.FinalizerName) {
		controllerutil.AddFinalizer(clone, constant.FinalizerName)
		err := r.Update(context.Background(), clone)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, r.BgpServer.HandleBgpPeer(clone, !matchNode)
}

func (r BgpPeerReconciler) Start(stopCh <-chan struct{}) error {
	go r.run(stopCh)

	return nil
}

func (r BgpPeerReconciler) run(stopCh <-chan struct{}) {
	t := time.NewTicker(30 * time.Second)

	for {
		select {
		case <-t.C:

			peers := &v1alpha2.BgpPeerList{}
			err := r.List(context.Background(), peers)
			if err != nil {
				continue
			}

			status := r.BgpServer.HandleBgpPeerStatus(peers.Items)

			//update status
			for _, peer := range peers.Items {
				clone := peer.DeepCopy()
				found := false

				for _, tmp := range status {
					if clone.Spec.Conf.NeighborAddress == tmp.Spec.Conf.NeighborAddress {
						clone.Status = tmp.Status
						found = true
						break
					}
				}
				if !found {
					delete(clone.Status.NodesPeerStatus, util.GetNodeName())
				}

				if !reflect.DeepEqual(clone.Status, peer.Status) {
					r.Status().Update(context.Background(), clone)
				}
			}

		case <-stopCh:
			return
		}
	}
}

func (r BgpPeerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.BgpPeer{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				old := e.ObjectOld.(*v1alpha2.BgpPeer)
				new := e.ObjectNew.(*v1alpha2.BgpPeer)
				if !reflect.DeepEqual(old.DeletionTimestamp, new.DeletionTimestamp) {
					return true
				}
				if !reflect.DeepEqual(old.Spec, new.Spec) {
					return true
				}
				return false

			},
		}).Complete(r)
}

func SetupBgpPeerReconciler(bgpServer *bgp.Bgp, mgr ctrl.Manager) error {
	bgpPeer := BgpPeerReconciler{
		Client:        mgr.GetClient(),
		BgpServer:     bgpServer,
		EventRecorder: mgr.GetEventRecorderFor("bgppeer"),
	}
	if err := bgpPeer.SetupWithManager(mgr); err != nil {
		return err
	}

	return mgr.Add(bgpPeer)
}
