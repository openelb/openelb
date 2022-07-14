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
	"fmt"
	"reflect"
	"time"

	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/metrics"
	"github.com/openelb/openelb/pkg/speaker/bgp"
	"github.com/openelb/openelb/pkg/util"
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

func peerMatchNode(peer *v1alpha2.BgpPeer, node *corev1.Node) (bool, error) {
	if peer.Spec.NodeSelector == nil {
		return true, nil
	}

	lbls := labels.Set(node.GetLabels())

	peerSelector, err := metav1.LabelSelectorAsSelector(peer.Spec.NodeSelector)
	if err != nil {
		return false, fmt.Errorf("BgpPeer %s spec.NodeSelector invalid, err=%v", peer.Name, err)
	}

	if peerSelector.Matches(lbls) {
		return true, nil
	}

	return false, nil
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

		matchNode, err = peerMatchNode(bgpPeer, node)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	clone := bgpPeer.DeepCopy()

	if util.IsDeletionCandidate(clone, constant.FinalizerName) {
		err := r.BgpServer.HandleBgpPeer(clone, true)
		if err != nil {
			log.Error(err, "cannot delete bgp peer, maybe need to delete manually")
		}

		controllerutil.RemoveFinalizer(clone, constant.FinalizerName)
		return ctrl.Result{}, r.Update(context.Background(), clone)
	}

	if util.NeedToAddFinalizer(clone, constant.FinalizerName) {
		controllerutil.AddFinalizer(clone, constant.FinalizerName)
		metrics.InitBGPPeerMetrics(clone.Spec.Conf.NeighborAddress, util.GetNodeName())
		err := r.Update(context.Background(), clone)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, r.BgpServer.HandleBgpPeer(clone, !matchNode)
}

func (r BgpPeerReconciler) Start(stopCh <-chan struct{}) error {
	err := r.CleanBgpPeerStatus()
	if err != nil {
		return err
	}

	go r.run(stopCh)

	return nil
}

func (r BgpPeerReconciler) updatePeerStatus() {
	peers := &v1alpha2.BgpPeerList{}
	err := r.List(context.Background(), peers)
	if err != nil {
		return
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
		r.BgpServer.UpdatePeerMetrics(&peer, !found)
	}
}

func (r BgpPeerReconciler) run(stopCh <-chan struct{}) {
	t := time.NewTicker(time.Duration(syncStatusPeriod) * time.Second)

	for {
		select {
		case <-t.C:
			r.updatePeerStatus()

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
				if util.DutyOfCNI(nil, e.Meta) {
					return false
				}
				return true
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldPeer := e.ObjectOld.(*v1alpha2.BgpPeer)
				newPeer := e.ObjectNew.(*v1alpha2.BgpPeer)
				if !util.DutyOfCNI(e.MetaOld, e.MetaNew) {
					if !reflect.DeepEqual(oldPeer.DeletionTimestamp, newPeer.DeletionTimestamp) {
						return true
					}
					if !reflect.DeepEqual(oldPeer.Spec, newPeer.Spec) {
						return true
					}
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

func (r *BgpPeerReconciler) CleanBgpPeerStatus() error {
	peers := &v1alpha2.BgpPeerList{}
	err := r.Client.List(context.Background(), peers)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	for _, peer := range peers.Items {
		clone := peer.DeepCopy()
		clone.Status = v1alpha2.BgpPeerStatus{}
		if reflect.DeepEqual(clone.Status, peer.Status) {
			continue
		}
		err = r.Client.Status().Update(context.Background(), &peer)
		if err != nil {
			return err
		}
	}

	return nil
}
