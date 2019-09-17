/*

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

package lb

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/constant"
	portererror "github.com/kubesphere/porter/pkg/errors"
	"github.com/kubesphere/porter/pkg/ipam"
	"github.com/kubesphere/porter/pkg/util"
	"github.com/kubesphere/porter/pkg/validate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"github.com/kubesphere/porter/pkg/route"
)

// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	IPAM *ipam.IPAM
	client.Client
	Log logr.Logger
	record.EventRecorder
	route.Advertiser 
}

func (r *ServiceReconciler) getNewerService(serv *corev1.Service) error {
	return r.Get(context.TODO(), types.NamespacedName{Namespace: serv.Namespace, Name: serv.Name}, serv)
}

func (r *ServiceReconciler) getNewerEIP(eip *v1alpha1.Eip) error {
	return r.Get(context.TODO(), types.NamespacedName{Name: eip.Name}, eip)
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	//service
	r.Client = mgr.GetClient()
	r.EventRecorder = mgr.GetEventRecorderFor("service")
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if validate.IsTypeLoadBalancer(e.ObjectOld) || validate.IsTypeLoadBalancer(e.ObjectNew) {
				if validate.HasPorterLBAnnotation(e.MetaNew.GetAnnotations()) || validate.HasPorterLBAnnotation(e.MetaOld.GetAnnotations()) {
					return e.ObjectOld != e.ObjectNew
				}
			}
			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			if validate.IsTypeLoadBalancer(e.Object) {
				return validate.HasPorterLBAnnotation(e.Meta.GetAnnotations())
			}
			return false
		},
	}
	// Watch for changes to Service
	//return ctl.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForObject{}, p)
	ctl, err := ctrl.NewControllerManagedBy(mgr).For(&corev1.Service{}).WithEventFilter(p).Named("LBController").Build(r)
	if err != nil {
		r.Log.Error(err, "Failed to build controller")
		return err
	}
	//endpoints
	p = predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			svc := &corev1.Service{}
			err := r.Get(context.TODO(), types.NamespacedName{Namespace: e.MetaOld.GetNamespace(), Name: e.MetaOld.GetName()}, svc)
			if err != nil {
				if !errors.IsNotFound(err) {
					r.Log.Error(err, "Service missing when watch Endpoints updating")
				}
				return false
			}
			if validate.IsTypeLoadBalancer(svc) && validate.HasPorterLBAnnotation(svc.GetAnnotations()) {
				old := e.ObjectOld.(*corev1.Endpoints)
				new := e.ObjectNew.(*corev1.Endpoints)
				return validate.IsNodeChangedWhenEndpointUpdated(old, new)
			}
			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			svc := &corev1.Service{}
			err := r.Get(context.TODO(), types.NamespacedName{Namespace: e.Meta.GetNamespace(), Name: e.Meta.GetName()}, svc)
			if err != nil {
				if !errors.IsNotFound(err) {
					r.Log.Error(err, "Something wrong when watch Endpoints creating")
				}
				return false
			}
			if validate.IsTypeLoadBalancer(svc) {
				return validate.HasPorterLBAnnotation(svc.GetAnnotations())
			}
			return false
		},
	}
	return ctl.Watch(&source.Kind{Type: &corev1.Endpoints{}}, &handler.EnqueueRequestForObject{}, p)
}

func (r *ServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	r.Log = r.Log.WithValues("porter", req.NamespacedName)
	// your logic here
	// Fetch the Service instance
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return r.reconcile(req.NamespacedName)
	})
	if err != nil {
		switch t := err.(type) {
		case portererror.ResourceNotEnoughError:
			r.Log.Info(t.Error() + ", waiting for requeue")
			return ctrl.Result{
				RequeueAfter: time.Second * 10,
			}, nil
		case portererror.EIPNotFoundError:
			r.Log.Info("Detect unknown ips in annotations")
			return ctrl.Result{}, r.clearAnnotation(req.NamespacedName)
		default:
			if errors.IsNotFound(err) {
				r.Log.Info("Maybe sevice has been deleted, skipping reconciling")
				return ctrl.Result{}, nil
			}
			r.Log.Error(t, "Create LB for service failed")
			return ctrl.Result{RequeueAfter: time.Second * 10}, t
		}
	}
	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) useFinalizerIfNeeded(serv *corev1.Service) (bool, error) {
	if serv.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object.
		// double check before appending finalizer ref: https://github.com/kubesphere/porter/issues/43
		if !validate.HasPorterLBAnnotation(serv.GetAnnotations()) {
			r.Log.Error(fmt.Errorf("service does not have porter annotation"), "Watching filter seems not take affect")
			return true, nil
		}
		if !util.ContainsString(serv.ObjectMeta.Finalizers, constant.FinalizerName) {
			serv.ObjectMeta.Finalizers = append(serv.ObjectMeta.Finalizers, constant.FinalizerName)
			if err := r.Update(context.Background(), serv); err != nil {
				r.Log.Info("Failed to use update to  append finalizer to service", "service", serv.Name)
				return false, err
			}
			r.Log.Info("Append Finalizer to service", "ServiceName", serv.Name, "Namespace", serv.Namespace)
		}
	} else {
		// The object is being deleted
		if util.ContainsString(serv.ObjectMeta.Finalizers, constant.FinalizerName) {
			// our finalizer is present, so lets handle our external dependency
			if err := r.deleteLB(serv); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return false, err
			}
			// remove our finalizer from the list and update it.
			serv.ObjectMeta.Finalizers = util.RemoveString(serv.ObjectMeta.Finalizers, constant.FinalizerName)
			if err := r.Update(context.Background(), serv); err != nil {
				if errors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			r.Log.Info("Remove Finalizer before service deleted", "ServiceName", serv.Name, "Namespace", serv.Namespace)
			return true, nil
		}
	}
	return false, nil
}

func (r *ServiceReconciler) clearAnnotation(key types.NamespacedName) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		serv := &corev1.Service{}
		err := r.Get(context.TODO(), key, serv)
		if err != nil {
			return err
		}
		delete(serv.Annotations, PorterEIPAnnotationKey)
		err = r.Update(context.Background(), serv)
		if err != nil {
			r.Log.Info("[Will Retry] Failed to clear Annotations")
			return err
		}
		return nil
	})
}
