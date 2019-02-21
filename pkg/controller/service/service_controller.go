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

package service

import (
	"context"
	"reflect"
	"time"

	portererror "github.com/kubesphere/porter/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */
var log = logf.Log.WithName("lb-controller")

// Add creates a new Service Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileService{Client: mgr.GetClient(), scheme: mgr.GetScheme(), EventRecorder: mgr.GetRecorder("manager")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("service-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}
	wm := NewWatchManager(c, mgr)
	return wm.AddAllWatch()
}

var _ reconcile.Reconciler = &ReconcileService{}

// ReconcileService reconciles a Service object
type ReconcileService struct {
	client.Client
	scheme *runtime.Scheme
	record.EventRecorder
}

// Reconcile reads that state of the cluster for a Service object and makes changes based on the state read
// and what is in the Service.Spec
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
func (r *ReconcileService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Service instance
	log.Info("Begin to reconclie for service")
	svc := &corev1.Service{}
	err := r.Get(context.TODO(), request.NamespacedName, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	origin := svc.DeepCopy()
	reconcileLog := log.WithValues("Service Name", svc.GetName(), "Namespace", svc.GetNamespace())
	needReconcile, err := r.useFinalizerIfNeeded(svc)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	if needReconcile {
		return reconcile.Result{}, nil
	}
	if len(svc.Status.LoadBalancer.Ingress) == 0 || !r.checkLB(svc) {
		err := r.createLB(svc)
		if err != nil {
			switch t := err.(type) {
			case portererror.ResourceNotEnoughError:
				reconcileLog.Info(t.Error() + ", waiting for requeue")
				return reconcile.Result{
					RequeueAfter: 15 * time.Second,
				}, nil
			case portererror.EIPNotFoundError:
				reconcileLog.Error(nil, "Detect non-exist ips in field 'ExternalIPs'")
				r.Event(svc, corev1.EventTypeWarning, "Detect non-exist externalIPs", "Clear field 'ExternalIPs' before using Porter")
				svc.Spec.ExternalIPs = []string{}
				err = r.Update(context.Background(), svc)
				if err != nil {
					reconcileLog.Error(nil, "Failed to clear field 'ExternalIPs'")
					return reconcile.Result{}, err
				}
				svc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{}
				err = r.Status().Update(context.Background(), svc)
				if err != nil {
					reconcileLog.Error(nil, "Failed to clear field 'LoadBalancer Ingress'")
					return reconcile.Result{}, err
				}
				return reconcile.Result{}, nil
			default:
				reconcileLog.Error(t, "Create LB for service failed")
				return reconcile.Result{}, t
			}
		}
	}
	if !reflect.DeepEqual(svc, origin) {
		err := r.Update(context.Background(), svc)
		if err != nil {
			reconcileLog.Error(nil, "update service instance failed")
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}
