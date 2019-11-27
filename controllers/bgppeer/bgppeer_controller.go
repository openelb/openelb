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

package bgppeer

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/kiali/kiali/log"
	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/bgpwrapper"
	"github.com/kubesphere/porter/pkg/constant"
	"github.com/kubesphere/porter/pkg/errors"
	"github.com/kubesphere/porter/pkg/util"
	k8serror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BgpPeerReconciler reconciles a BgpPeer object
type BgpPeerReconciler struct {
	client.Client
	Log logr.Logger
	bgpwrapper.Interface
	SyncInterval time.Duration
}

func (r *BgpPeerReconciler) SyncPeersState() {
	time.Sleep(r.SyncInterval)
	r.syncPeersState()
}

func (r *BgpPeerReconciler) syncPeersState() {
	peers := &networkv1alpha1.BgpPeerList{}
	err := r.List(context.TODO(), peers)
	if err != nil {
		r.Log.Error(err, "Failed to list peers")
		return
	}
	for _, peer := range peers.Items {
		err = r.syncPeer(&peer)
		if err != nil {
			r.Log.Error(err, "Failed to sync peer status", "Name", peer.Name, "Address", peer.Spec.Conf.NeighborAddress)
		}
	}
}

func (r *BgpPeerReconciler) syncPeer(p *networkv1alpha1.BgpPeer) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		gbPeer, err := r.GetPeer(p)
		if err != nil {
			return err
		}
		p.Status.SessionState = networkv1alpha1.SessionState(gbPeer.State.SessionState)
		uptime, err := util.ParseProtoTime(gbPeer.Timers.State.Uptime)
		if err != nil {
			return err
		}
		p.Status.Uptime = *uptime
		downtime, err := util.ParseProtoTime(gbPeer.Timers.State.Downtime)
		if err != nil {
			return err
		}
		p.Status.Uptime = *downtime
		//TODO: add more fields here
		return r.Status().Update(context.Background(), p)
	})
}

// +kubebuilder:rbac:groups=network.kubesphere.io,resources=bgppeers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.kubesphere.io,resources=bgppeers/status,verbs=get;update;patch

func (r *BgpPeerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("bgppeer", req.NamespacedName)
	log.Info("-------------Reconcile Beginning--------------")
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		peer := &networkv1alpha1.BgpPeer{}
		err := r.Get(context.TODO(), req.NamespacedName, peer)
		if err != nil {
			if k8serror.IsNotFound(err) {
				return nil
			}
			log.Info("Failed to get object")
			return err
		}
		deleted, err := r.reconcileFinalizier(peer)
		if err != nil {
			return err
		}
		if deleted {
			return nil
		}
		return r.ensurePeer(peer)
	})
	log.Info("-------------Reconcile Ended--------------")
	return ctrl.Result{}, err
}

func (r *BgpPeerReconciler) ensurePeer(peer *networkv1alpha1.BgpPeer) error {
	log := r.Log.WithValues(peer.Name, "Address", peer.Spec.Conf.NeighborAddress)
	_, err := r.GetPeer(peer)
	if err != nil {
		if !errors.IsResourceNotFound(err) {
			log.Info("Failed to get peer in BPGServer")
			return err
		}
		err = r.AddPeer(peer)
		if err != nil {
			log.Info("Failed to add peer to BPGServer")
			return err
		}
		log.Info("Peer added")
		return nil
	}
	if need, err := r.NeedUpdate(peer); err == nil {
		if need {
			log.Info("Detect changes, updating peer")
			err = r.UpdatePeer(peer)
			if err != nil {
				log.Info("Failed to update peer")
				return err
			}
			log.Info("Peer updated successfully")
			return nil
		}
	} else {
		log.Info("Failed to get state of bgp peer")
		return err
	}
	return nil
}

func (r *BgpPeerReconciler) ensurePeerDeleted(peer *networkv1alpha1.BgpPeer) error {
	_, err := r.GetPeer(peer)
	if err != nil {
		if !errors.IsResourceNotFound(err) {
			log.Info("Failed to get peer in BPGServer")
			return err
		}
		return nil
	}
	return r.DeletePeer(peer)
}

func (r *BgpPeerReconciler) reconcileFinalizier(peer *networkv1alpha1.BgpPeer) (bool, error) {
	log := r.Log.WithValues(peer.Name, "Address", peer.Spec.Conf.NeighborAddress)
	finalizer := "peer." + constant.FinalizerName
	if peer.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object.
		if !util.ContainsString(peer.ObjectMeta.Finalizers, finalizer) {
			peer.ObjectMeta.Finalizers = append(peer.ObjectMeta.Finalizers, finalizer)
			if err := r.Update(context.Background(), peer); err != nil {
				return false, err
			}
			log.Info("Append Finalizer to peer", "peerName", peer.Name)
			return false, nil
		}
	} else {
		// The object is being deleted
		if util.ContainsString(peer.ObjectMeta.Finalizers, finalizer) {
			log.Info("Begin to remove finalizer")
			// our finalizer is present, so lets handle our external dependency
			// remove our finalizer from the list and update it.
			err := r.ensurePeerDeleted(peer)
			if err != nil {
				log.Info("Failed to del peer", "peerName", peer.Name, "Address", peer.Spec.Conf.NeighborAddress)
				return false, err
			}
			peer.ObjectMeta.Finalizers = util.RemoveString(peer.ObjectMeta.Finalizers, finalizer)
			if err := r.Update(context.Background(), peer); err != nil {
				if k8serror.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			log.Info("Remove finalizer before peer deleted", "peerName", peer.Name, "Address", peer.Spec.Conf.NeighborAddress)
			return true, nil
		}
	}
	return false, nil
}

func (r *BgpPeerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkv1alpha1.BgpPeer{}).
		Complete(r)
}
