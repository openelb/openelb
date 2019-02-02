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

	"github.com/kubesphere/porter/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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

const (
	FinalizerName string = "finalizer.lb.kubesphere.io/v1apha1"
)

// Add creates a new Service Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileService{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
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
}

// Reconcile reads that state of the cluster for a Service object and makes changes based on the state read
// and what is in the Service.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
func (r *ReconcileService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Service instance
	log.Info("Begin to reconclie for service")
	instance := &corev1.Service{}
	origin := instance.DeepCopy()
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	needReconile, err := r.useFinalizerIfNeeded(instance)
	if err != nil {
		return reconcile.Result{Requeue: true}, err
	}
	if needReconile {
		return reconcile.Result{}, nil
	}
	if len(instance.Status.LoadBalancer.Ingress) == 0 {
		err := r.createLB(instance)
		if err != nil {
			log.Error(err, "Create LB for service failed", "Service Name", instance.GetName())
			return reconcile.Result{}, err
		}
		instance.Status.LoadBalancer.Ingress = append(instance.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{
			IP: instance.Spec.ExternalIPs[0],
		})
	} else {
		if !r.checkLB(instance) {
			log.Info("Detect ingress IP, however no route exsit in gbp, maybe due to the restart of controller")
			err = r.createLB(instance)
			if err != nil {
				log.Error(err, "Create LB for service failed", "Service Name", instance.GetName())
				return reconcile.Result{}, err
			}
		}
	}
	if !reflect.DeepEqual(instance.Status, origin.Status) {
		r.Client.Status().Update(context.Background(), instance)
	}
	return reconcile.Result{}, nil
}

func (r *ReconcileService) useFinalizerIfNeeded(serv *corev1.Service) (bool, error) {
	if serv.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object.
		if !util.ContainsString(serv.ObjectMeta.Finalizers, FinalizerName) {
			serv.ObjectMeta.Finalizers = append(serv.ObjectMeta.Finalizers, FinalizerName)
			if err := r.Update(context.Background(), serv); err != nil {
				return false, err
			}
			log.Info("Append Finalizer to service", "ServiceName", serv.Name, "Namespace", serv.Namespace)
			return true, nil
		}
	} else {
		// The object is being deleted
		if util.ContainsString(serv.ObjectMeta.Finalizers, FinalizerName) {
			// our finalizer is present, so lets handle our external dependency
			if err := r.deleteLB(serv); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return false, err
			}

			// remove our finalizer from the list and update it.
			serv.ObjectMeta.Finalizers = util.RemoveString(serv.ObjectMeta.Finalizers, FinalizerName)
			if err := r.Update(context.Background(), serv); err != nil {
				return true, nil
			}
			log.Info("Remove Finalizer before service deleted", "ServiceName", serv.Name, "Namespace", serv.Namespace)
		}
	}
	return false, nil
}
