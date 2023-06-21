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

	"github.com/openelb/openelb/api/v1alpha2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BgpConfReconciler reconciles a BgpConf object
type EIPReconciler struct {
	client.Client
	record.EventRecorder

	Handler func(*v1alpha2.Eip) error
}

func (e *EIPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.Eip{}).
		Named("EIPController").
		Complete(e)
}

//+kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster BgpConf CRD closer to the desired state.
func (l *EIPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.Log.WithValues("eip", req.NamespacedName.Name)
	log.V(1).Info("start setup openelb eip")
	defer log.V(1).Info("finish reconcile openelb eip")

	eip := &v1alpha2.Eip{}
	if err := l.Client.Get(ctx, req.NamespacedName, eip); err != nil {
		if errors.IsNotFound(err) {
			log.Info("NotFound", " req.NamespacedName", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, l.Handler(eip)
}
