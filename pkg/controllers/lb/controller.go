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
	"math/rand"
	"reflect"

	"github.com/kubesphere/porterlb/api/v1alpha2"
	"github.com/kubesphere/porterlb/pkg/constant"
	"github.com/kubesphere/porterlb/pkg/controllers/ipam"
	"github.com/kubesphere/porterlb/pkg/util"
	"github.com/kubesphere/porterlb/pkg/validate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
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

	return IsPorterService(svc)
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return IsPorterService(e.ObjectNew)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return IsPorterService(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return IsPorterService(e.Object)
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
			return r.shouldReconcileEP(e.MetaNew)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return r.shouldReconcileEP(e.Meta)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.shouldReconcileEP(e.Meta)
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
			if nodeReady(e.ObjectOld) != nodeReady(e.ObjectNew) {
				return true
			} else {
				return false
			}
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}
	return ctl.Watch(&source.Kind{Type: &corev1.Node{}}, &EnqueueRequestForNode{Client: r.Client}, np)
}

func (r *ServiceReconciler) callSetLoadBalancer(result ipam.IPAMResult, svc *corev1.Service) error {
	nodes, err := r.getServiceNodes(svc)
	if err != nil {
		return err
	}

	svcIP := result.Addr

	var announceNodes []corev1.Node
	if result.Protocol == constant.PorterProtocolLayer2 {
		if len(nodes) == 0 {
			return result.Sp.DelBalancer(svcIP)
		}

		index := rand.Int() % len(nodes)
		found := false
		preNode, ok := svc.Annotations[constant.PorterLayer2Annotation]
		if ok {
			for i, node := range nodes {
				if node.Name == preNode {
					index = i
					found = true
					break
				}
			}
		}

		if !found {
			if svc.Annotations == nil {
				svc.Annotations = make(map[string]string)
			}
			svc.Annotations[constant.PorterLayer2Annotation] = nodes[index].Name

			err = r.Update(context.Background(), svc)
			if err != nil {
				return err
			}
		}

		announceNodes = append(announceNodes, nodes[index])
	} else {
		announceNodes = append(announceNodes, nodes...)
	}

	return result.Sp.SetBalancer(svcIP, announceNodes)
}

func (r *ServiceReconciler) callDelLoadBalancer(result ipam.IPAMResult, svc *corev1.Service) error {
	if result.Addr != "" {
		if svc.Annotations != nil && svc.Annotations[constant.PorterLayer2Annotation] != "" {
			delete(svc.Annotations, constant.PorterLayer2Annotation)
			err := r.Update(context.Background(), svc)
			if err != nil {
				return err
			}
		}
		return result.Sp.DelBalancer(result.Addr)
	}
	return nil
}

const (
	ReasonDeleteLoadBalancer = "deleteLoadBalancer"
	ReasonAddLoadBalancer    = "addLoadBalancer"
	AddLoadBalancerMsg       = "success to add nexthops %v"
	AddLoadBalancerFailedMsg = "failed to add nexthops %v, err=%v"
	DelLoadBalancerMsg       = "loadbalancer ip changed from %s to %s"
	DelLoadBalancerFailedMsg = "speaker del loadbalancer failed, err=%v"
)

func (r *ServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	var (
		result ipam.IPAMResult
	)

	log := ctrl.Log.WithValues("service", req.NamespacedName)
	log.Info("setup porter service")

	svc := &corev1.Service{}
	err := r.Get(context.TODO(), req.NamespacedName, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	args := r.constructIPAMArgs(svc)
	result, err = ipam.IPAMAllocator.UnAssignIP(args, true)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Check if the IP address specified by the service should be changed.
	if args.ShouldUnAssignIP(result) {
		err = r.callDelLoadBalancer(result, svc)
		if err != nil {
			r.Event(svc, corev1.EventTypeWarning, ReasonDeleteLoadBalancer, fmt.Sprintf(DelLoadBalancerFailedMsg, err))
			return ctrl.Result{}, err
		}
		_, err = ipam.IPAMAllocator.UnAssignIP(args, false)
		if err != nil {
			return ctrl.Result{}, err
		}

		result.Clean()
	}

	if !args.Unalloc {
		if !result.Assigned() {
			result, err = ipam.IPAMAllocator.AssignIP(args)
			if err != nil {
				r.updateServiceEipInfo(result, svc)
				return ctrl.Result{}, err
			}
		}

		err = r.callSetLoadBalancer(result, svc)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, r.updateServiceEipInfo(result, svc)
}

func (r *ServiceReconciler) updateServiceEipInfo(result ipam.IPAMResult, svc *corev1.Service) error {
	clone := svc.DeepCopy()

	//update eip labels and annotations
	if clone.Labels == nil {
		clone.Labels = make(map[string]string)
	}
	if result.Assigned() {
		if !util.ContainsString(clone.Finalizers, constant.FinalizerName) {
			controllerutil.AddFinalizer(clone, constant.FinalizerName)
		}
		clone.Labels[constant.PorterEIPAnnotationKeyV1Alpha2] = result.Eip
	} else {
		controllerutil.RemoveFinalizer(clone, constant.FinalizerName)
		delete(clone.Labels, constant.PorterEIPAnnotationKeyV1Alpha2)
	}
	if !reflect.DeepEqual(svc.Labels, clone.Labels) {
		err := r.Update(context.Background(), clone)
		if err != nil {
			return err
		}
	}

	//update ingress status
	clone.Status.LoadBalancer.Ingress = nil
	if result.Assigned() {
		clone.Status.LoadBalancer.Ingress = append(clone.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{
			IP: result.Addr,
		})
	}
	if !reflect.DeepEqual(svc.Status, clone.Status) {
		err := r.Status().Update(context.Background(), clone)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *ServiceReconciler) constructIPAMArgs(svc *corev1.Service) ipam.IPAMArgs {
	args := ipam.IPAMArgs{
		Unalloc: true,
	}

	args.Key = types.NamespacedName{
		Name:      svc.Name,
		Namespace: svc.Namespace,
	}.String()

	if svc.Annotations != nil {
		if _, ok := svc.Annotations[constant.PorterAnnotationKey]; ok &&
			svc.Spec.Type == corev1.ServiceTypeLoadBalancer &&
			svc.DeletionTimestamp == nil {
			args.Unalloc = false
		}

		if ip, ok := svc.Annotations[constant.PorterEIPAnnotationKey]; ok {
			args.Addr = ip
		}

		if eip, ok := svc.Annotations[constant.PorterEIPAnnotationKeyV1Alpha2]; ok {
			args.Eip = eip
		}

		if protocol, ok := svc.Annotations[constant.PorterProtocolAnnotationKey]; ok {
			args.Protocol = protocol
		} else {
			args.Protocol = constant.PorterProtocolBGP
		}
	}

	if svc.Spec.LoadBalancerIP != "" {
		args.Addr = svc.Spec.LoadBalancerIP
	}

	return args
}

// The caller should check if the slice is empty.
func (r *ServiceReconciler) getServiceNodes(svc *corev1.Service) ([]corev1.Node, error) {
	//1. filter endpoints
	endpoints := &corev1.Endpoints{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: svc.GetNamespace(), Name: svc.GetName()}, endpoints)
	if err != nil {
		return nil, err
	}

	active := make(map[string]bool)
	for _, subnet := range endpoints.Subsets {
		for _, addr := range subnet.Addresses {
			active[*addr.NodeName] = true
		}
	}
	if len(active) <= 0 {
		return nil, nil
	}

	//2. get next hops
	nodeList := &corev1.NodeList{}
	err = r.List(context.TODO(), nodeList)
	if err != nil {
		return nil, err
	}

	resultNodes := make([]corev1.Node, 0)
	if svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal {
		for _, node := range nodeList.Items {
			if active[node.Name] {
				resultNodes = append(resultNodes, node)
			}
		}
	} else {
		for _, node := range nodeList.Items {
			if nodeReady(&node) {
				resultNodes = append(resultNodes, node)
			}
		}
	}

	return resultNodes, nil
}

func SetupServiceReconciler(mgr ctrl.Manager) error {
	lb := &ServiceReconciler{
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("PorterLB Manager"),
	}
	err := lb.SetupWithManager(mgr)
	return err
}

func IsPorterService(obj runtime.Object) bool {
	if svc, ok := obj.(*corev1.Service); ok {
		if svc.Labels != nil {
			if _, ok := svc.Labels[constant.PorterEIPAnnotationKeyV1Alpha2]; ok {
				return true
			}
		}

		return validate.HasPorterLBAnnotation(svc.Annotations) && validate.IsTypeLoadBalancer(svc)
	}
	return false
}
