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

package speaker

import (
	"context"
	"time"

	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// BgpConfReconciler reconciles a BgpConf object
type EIPReconciler struct {
	client.Client
	record.EventRecorder

	Reload   chan event.GenericEvent
	Reloader func(context.Context) error
	Handler  func(context.Context, *v1alpha2.Eip) error
}

func (e *EIPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.Eip{}).
		WatchesRawSource(&source.Channel{Source: e.Reload}, &handler.EnqueueRequestForObject{}).
		Named("EIPController").
		Complete(e)
}

//+kubebuilder:rbac:groups=network.kubesphere.io,resources=eips,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=network.kubesphere.io,resources=eips/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster BgpConf CRD closer to the desired state.
func (e *EIPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).Infof("Starting to sync eip %s", req.Name)
	startTime := time.Now()

	defer func() {
		klog.V(4).Infof("Finished syncing eip %s in %s", req.Name, time.Since(startTime))
	}()

	if e.reloadLayer2Speaker(req) {
		return ctrl.Result{}, e.Reloader(ctx)
	}

	eip := &v1alpha2.Eip{}
	if err := e.Client.Get(ctx, req.NamespacedName, eip); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, e.Handler(ctx, eip)
}

func (e *EIPReconciler) reloadLayer2Speaker(req ctrl.Request) bool {
	return req.Name == constant.Layer2ReloadEIPName && req.Namespace == constant.Layer2ReloadEIPNamespace
}
