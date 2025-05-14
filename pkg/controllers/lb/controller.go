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
	"time"

	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/controllers/ipam"
	"github.com/openelb/openelb/pkg/util"
	"github.com/openelb/openelb/pkg/validate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "EIPController"

	ReasonDeleteLoadBalancer = "deleteLoadBalancer"
	ReasonAddLoadBalancer    = "addLoadBalancer"
	AddLoadBalancerMsg       = "success to add nexthops %v"
	AddLoadBalancerFailedMsg = "failed to add nexthops %v, err=%v"
	DelLoadBalancerMsg       = "loadbalancer ip changed from %s to %s"
	DelLoadBalancerFailedMsg = "speaker del loadbalancer failed, err=%v"
)

// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get;update;patch

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	ipmanager *ipam.Manager
	record.EventRecorder
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

	eipfun := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldEip := e.ObjectOld.(*v1alpha2.Eip)
			newEip := e.ObjectNew.(*v1alpha2.Eip)
			emptyStatus := v1alpha2.EipStatus{}

			return reflect.DeepEqual(oldEip.Status, emptyStatus) && !reflect.DeepEqual(newEip.Status, emptyStatus)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}
	err = ctl.Watch(source.Kind(mgr.GetCache(), &v1alpha2.Eip{}), &EnqueueRequestForNode{Client: r.Client}, eipfun)
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
	err = ctl.Watch(source.Kind(mgr.GetCache(), &corev1.Node{}), &EnqueueRequestForNode{Client: r.Client}, np)
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
	err = ctl.Watch(source.Kind(mgr.GetCache(), &appsv1.Deployment{}), &EnqueueRequestForDeAndDs{Client: r.Client}, dedsp)
	if err != nil {
		return err
	}
	err = ctl.Watch(source.Kind(mgr.GetCache(), &appsv1.DaemonSet{}), &EnqueueRequestForDeAndDs{Client: r.Client}, dedsp)
	if err != nil {
		return err
	}
	return mgr.GetFieldIndexer().IndexField(context.TODO(), &corev1.Pod{}, "status.phase", func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{string(pod.Status.Phase)}
	})
}

func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).Infof("Starting to setup service %s/%s", req.Namespace, req.Name)
	startTime := time.Now()

	defer func() {
		klog.V(4).Infof("Finished reconcile service %s/%s in %s", req.Namespace, req.Name, time.Since(startTime))
	}()

	svc := &corev1.Service{}
	err := r.Get(ctx, req.NamespacedName, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Reconcile by OpenELB NodeProxy if this service is specified to be exported by it
	if validate.HasOpenELBNPAnnotation(svc.Annotations) {
		return r.reconcileNP(ctx, svc)
	}

	request, err := r.ipmanager.ConstructRequest(ctx, svc)
	if err != nil {
		return ctrl.Result{}, err
	}

	if request.Release == nil && request.Allocate == nil {
		return ctrl.Result{}, nil
	}

	clone := svc.DeepCopy()
	statusIPs := svc.Status.LoadBalancer.Ingress
	if request.Release != nil {
		klog.V(4).Infof("Release service loadbalanceip %s", request.Release.String())
		err = r.ipmanager.ReleaseIP(ctx, request.Release)
		if err != nil {
			klog.Errorf("%s release ip[%s] form eip[%s] error :%s", request.Release.Key, request.Release.IP, request.Release.Eip, err.Error())
			r.Event(svc, corev1.EventTypeWarning, "ReleaseIPFailed", err.Error())
			return ctrl.Result{}, err
		}

		//clean node proxy data
		if result, err := r.cleanNodeProxyData(ctx, clone); err != nil {
			return result, err
		}

		//update service
		statusIPs = []corev1.LoadBalancerIngress{}
		controllerutil.RemoveFinalizer(clone, constant.FinalizerName)
		delete(clone.Labels, constant.OpenELBEIPAnnotationKeyV1Alpha2)
		r.Eventf(svc, corev1.EventTypeNormal, "ReleaseIP", "success to release ip: %s", request.Release.IP)
		klog.Infof("release ip[%s] from eip[%s] for service %s successfully", request.Release.IP, request.Release.Eip, request.Release.Key)
	}

	if request.Allocate != nil {
		klog.V(4).Infof("Allocate service loadbalanceip %s", request.Allocate.String())
		err = r.ipmanager.AssignIP(ctx, svc.Spec.IPFamilies, request.Allocate)
		if err != nil {
			klog.Errorf("%s assign ip[%s] form eip[%s] error :%s", request.Allocate.Key, request.Allocate.IP, request.Allocate.Eip, err.Error())
			r.Event(svc, corev1.EventTypeWarning, "AssignIPFailed", err.Error())
			clone.Status.LoadBalancer.Ingress = statusIPs
			r.updateReconcileResult(ctx, svc, clone)
			return ctrl.Result{}, err
		}

		//update service
		if !util.ContainsString(clone.Finalizers, constant.FinalizerName) {
			controllerutil.AddFinalizer(clone, constant.FinalizerName)
		}
		if clone.Labels == nil {
			clone.Labels = make(map[string]string)
		}
		clone.Labels[constant.OpenELBEIPAnnotationKeyV1Alpha2] = request.Allocate.Eip
		statusIPs = []corev1.LoadBalancerIngress{{IP: request.Allocate.IP}}
		r.Eventf(svc, corev1.EventTypeNormal, "AssignIP", "success to assign ip: %s", request.Allocate.IP)
		klog.Infof("assign ip[%s] from eip[%s] for service %s successfully", request.Allocate.IP, request.Allocate.Eip, request.Allocate.Key)
	}

	clone.Status.LoadBalancer.Ingress = statusIPs
	return ctrl.Result{}, r.updateReconcileResult(ctx, svc, clone)
}

// updateReconcileResult update service resource and status
func (r *ServiceReconciler) updateReconcileResult(ctx context.Context, svc, resultSvc *corev1.Service) error {
	clone := resultSvc.DeepCopy()
	if !reflect.DeepEqual(svc.Labels, resultSvc.Labels) {
		if err := r.Update(ctx, clone); err != nil {
			klog.Errorf("update update labels error:%s", err.Error())
			return err
		}
	}

	if !reflect.DeepEqual(svc.Status, resultSvc.Status) {
		clone.Status = resultSvc.Status
		if err := r.Status().Update(context.Background(), clone); err != nil {
			klog.Errorf("update service status error:%s", err.Error())
			r.Event(svc, corev1.EventTypeWarning, "UpdateServiceStatus", err.Error())
			return err
		}
	}

	return nil
}

func SetupServiceReconciler(mgr ctrl.Manager) error {
	lb := &ServiceReconciler{
		ipmanager:     ipam.NewManager(mgr.GetClient()),
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("OpenELBController"),
	}
	lb.ipmanager.EventRecorder = lb.EventRecorder
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
