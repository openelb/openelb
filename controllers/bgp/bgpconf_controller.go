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

// BgpConfReconciler reconciles a BgpConf object
type BgpConfReconciler struct {
	client.Client
	Log       logr.Logger
	BgpServer *bgpserver.BgpServer
}

// +kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs/status,verbs=get;update;patch

func (r *BgpConfReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("bgpconf", req.NamespacedName)

	instance := &networkv1alpha1.BgpConf{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	deleted, err := r.useFinalizerIfNeeded(instance)
	if deleted {
		return ctrl.Result{}, nil
	}

	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}

	return ctrl.Result{}, r.BgpServer.HandleBgpGlobalConfig(&instance.Spec, false)
}

func (r *BgpConfReconciler) useFinalizerIfNeeded(conf *networkv1alpha1.BgpConf) (bool, error) {
	if conf.ObjectMeta.DeletionTimestamp.IsZero() {
		if !util.ContainsString(conf.ObjectMeta.Finalizers, constant.FinalizerName) {
			conf.ObjectMeta.Finalizers = append(conf.ObjectMeta.Finalizers, constant.FinalizerName)
			if err := r.Update(context.Background(), conf); err != nil {
				r.Log.Info("Failed to use update to  append finalizer to BgpConf", "service", conf.Name)
				return false, err
			}
			r.Log.Info("Append Finalizer to BgpConf", "ServiceName", conf.Name, "Namespace", conf.Namespace)
		}
	} else {
		// The object is being deleted
		if util.ContainsString(conf.ObjectMeta.Finalizers, constant.FinalizerName) {
			if err := r.BgpServer.HandleBgpGlobalConfig(&conf.Spec, true); err != nil {
				return false, err
			}

			// remove our finalizer from the list and update it.
			conf.ObjectMeta.Finalizers = util.RemoveString(conf.ObjectMeta.Finalizers, constant.FinalizerName)
			if err := r.Update(context.Background(), conf); err != nil {
				if errors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			r.Log.Info("Remove Finalizer before service deleted", "ServiceName", conf.Name, "Namespace", conf.Namespace)
			return true, nil
		}
	}
	return false, nil
}

func (r *BgpConfReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&networkv1alpha1.BgpConf{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return true
			},
		}).Complete(r)
}
