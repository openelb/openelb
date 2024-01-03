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
	"reflect"

	"github.com/go-logr/logr"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/controllers/ipam"
	"github.com/openelb/openelb/pkg/util"
	"github.com/openelb/openelb/pkg/validate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ReasonDeleteLoadBalancer = "deleteLoadBalancer"
	ReasonAddLoadBalancer    = "addLoadBalancer"
	AddLoadBalancerMsg       = "success to add nexthops %v"
	AddLoadBalancerFailedMsg = "failed to add nexthops %v, err=%v"
	DelLoadBalancerMsg       = "loadbalancer ip changed from %s to %s"
	DelLoadBalancerFailedMsg = "speaker del loadbalancer failed, err=%v"
)

// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=services/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	ipmanager *ipam.Manager
	log       logr.Logger
	record.EventRecorder
}

func (r *ServiceReconciler) shouldReconcileEP(e metav1.Object) bool {
	if e.GetAnnotations() != nil {
		if e.GetAnnotations()["control-plane.alpha.kubernetes.io/leader"] != "" {
			return false
		}
	}

	svc := &corev1.Service{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: e.GetNamespace(), Name: e.GetName()}, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			return false
		}
		return true
	}

	return IsOpenELBService(svc)
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return IsOpenELBService(e.ObjectNew)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return IsOpenELBService(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return IsOpenELBService(e.Object)
		},
	}

	ctl, err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		WithEventFilter(p).
		Named("LBController").
		Build(r)
	if err != nil {
		return err
	}

	//endpoints
	ep := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return r.shouldReconcileEP(e.ObjectNew)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return r.shouldReconcileEP(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.shouldReconcileEP(e.Object)
		},
	}
	err = ctl.Watch(&source.Kind{Type: &corev1.Endpoints{}}, &handler.EnqueueRequestForObject{}, ep)
	if err != nil {
		return err
	}

	bp := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			old := e.ObjectOld.(*v1alpha2.BgpConf)
			new := e.ObjectNew.(*v1alpha2.BgpConf)

			if !reflect.DeepEqual(old.Annotations, new.Annotations) {
				return true
			}

			return false
		},
	}
	err = ctl.Watch(&source.Kind{Type: &v1alpha2.BgpConf{}}, &EnqueueRequestForNode{Client: r.Client}, bp)
	if err != nil {
		return err
	}

	np := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if util.NodeReady(e.ObjectOld) != util.NodeReady(e.ObjectNew) {
				return true
			}
			if nodeAddrChange(e.ObjectOld, e.ObjectNew) {
				return true
			}
			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}
	err = ctl.Watch(&source.Kind{Type: &corev1.Node{}}, &EnqueueRequestForNode{Client: r.Client}, np)
	if err != nil {
		return err
	}

	// If there's any Service be deployed by OpenELB NodeProxy, controller will create Deployment or DaemonSet for Proxy Pod
	// If the status of such Deployment or DaemonSet changed, all OpenELB NodeProxy should be reconciled
	dedsp := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Maybe Deployment or DaemonSet was modified, so both should be looked at
			return r.shouldReconcileDeDs(e.ObjectNew) || r.shouldReconcileDeDs(e.ObjectOld)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return r.shouldReconcileDeDs(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.shouldReconcileDeDs(e.Object)
		},
	}
	err = ctl.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &EnqueueRequestForDeAndDs{Client: r.Client}, dedsp)
	if err != nil {
		return err
	}
	err = ctl.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &EnqueueRequestForDeAndDs{Client: r.Client}, dedsp)
	if err != nil {
		return err
	}
	return mgr.GetFieldIndexer().IndexField(context.TODO(), &corev1.Pod{}, "status.phase", func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{string(pod.Status.Phase)}
	})
}

func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.Log.WithValues("service", req.NamespacedName)
	log.Info("start setup openelb service")
	defer log.Info("finish reconcile openelb service")

	svc := &corev1.Service{}
	err := r.Get(ctx, req.NamespacedName, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	clone := svc.DeepCopy()
	// Reconcile by OpenELB NodeProxy if this service is specified to be exported by it
	if validate.HasOpenELBNPAnnotation(clone.Annotations) {
		return r.reconcileNP(clone)
	}

	request, err := r.ipmanager.ConstructRequest(ctx, clone)
	if err != nil || request == nil {
		return ctrl.Result{}, err
	}

	if request.Release == nil && request.Allocate == nil {
		return ctrl.Result{}, nil
	}

	if request.Release != nil {
		err = r.ipmanager.ReleaseIP(ctx, request.Release, clone)
		if err != nil {
			log.Error(err, "release ip", "request", request)
			return ctrl.Result{}, err
		}
	}

	if request.Allocate != nil {
		err = r.ipmanager.AssignIP(ctx, request.Allocate, clone)
		if err != nil {
			log.Error(err, "assign ip", "request", request)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, r.updateService(ctx, svc, clone)
}

func (r *ServiceReconciler) updateService(ctx context.Context, svc, clone *corev1.Service) error {
	update := false
	if !reflect.DeepEqual(svc.Labels, clone.Labels) {
		err := r.Update(ctx, clone)
		update = true
		if err != nil {
			return err
		}
	}

	if !reflect.DeepEqual(svc.Status, clone.Status) {
		if update {
			err := r.Get(ctx, types.NamespacedName{
				Namespace: svc.Namespace,
				Name:      svc.Name,
			}, svc)
			if err != nil {
				if errors.IsNotFound(err) {
					return nil
				}
				return err
			}

			clone.ObjectMeta = svc.ObjectMeta
		}

		return r.Status().Update(ctx, clone)
	}
	return nil
}

func SetupServiceReconciler(mgr ctrl.Manager) error {
	lb := &ServiceReconciler{
		ipmanager:     ipam.NewManager(mgr.GetClient()),
		Client:        mgr.GetClient(),
		log:           ctrl.Log.WithName("Manager"),
		EventRecorder: mgr.GetEventRecorderFor("Manager"),
	}
	return lb.SetupWithManager(mgr)
}

func IsOpenELBService(obj runtime.Object) bool {
	if svc, ok := obj.(*corev1.Service); ok {
		if svc.Labels != nil {
			if _, ok := svc.Labels[constant.OpenELBEIPAnnotationKeyV1Alpha2]; ok {
				return true
			}
		}

		return validate.HasOpenELBAnnotation(svc.Annotations) && validate.IsTypeLoadBalancer(svc)
	}
	return false
}
