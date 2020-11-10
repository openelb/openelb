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
	"github.com/projectcalico/libcalico-go/lib/set"
	"math/rand"
	"reflect"

	"github.com/kubesphere/porter/pkg/constant"
	"github.com/kubesphere/porter/pkg/controllers/ipam"
	"github.com/kubesphere/porter/pkg/util"
	"github.com/kubesphere/porter/pkg/validate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

func getActiveEndpointNode(ep *corev1.Endpoints) set.Set {
	active := make([]string, 0)
	for _, subnet := range ep.Subsets {
		for _, addr := range subnet.Addresses {
			active = append(active, *addr.NodeName)
		}
	}
	if len(active) <= 0 {
		return nil
	}
	return set.FromArray(active)
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return validate.IsPorterService(e.ObjectNew)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return validate.IsPorterService(e.Object)
		},
	}

	// Watch for changes to Service
	//return ctl.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForObject{}, p)
	ctl, err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		WithEventFilter(p).
		Named("LBController").
		Build(r)
	if err != nil {
		return err
	}

	//endpoints
	p = predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			svc := &corev1.Service{}
			err := r.Get(context.TODO(), types.NamespacedName{Namespace: e.MetaOld.GetNamespace(), Name: e.MetaOld.GetName()}, svc)
			if err != nil {
				return true
			}

			if !validate.IsPorterService(svc) {
				return false
			}

			oldSet := getActiveEndpointNode(e.ObjectOld.(*corev1.Endpoints))
			newSet := getActiveEndpointNode(e.ObjectNew.(*corev1.Endpoints))
			if newSet == nil && oldSet == nil {
				return false
			} else if newSet == nil || oldSet == nil {
				return true
			}
			return !oldSet.Equals(newSet)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			svc := &corev1.Service{}
			err := r.Get(context.TODO(), types.NamespacedName{Namespace: e.Meta.GetNamespace(), Name: e.Meta.GetName()}, svc)
			if err != nil {
				return true
			}
			return validate.IsPorterService(svc)
		},
	}
	return ctl.Watch(&source.Kind{Type: &corev1.Endpoints{}}, &handler.EnqueueRequestForObject{}, p)
}

func (r *ServiceReconciler) callSetLoadBalancer(result ipam.IPAMResult, svc *corev1.Service) ([]string, error) {
	nodesIP, err := r.getServiceNodes(svc)
	if err != nil {
		return nil, err
	}

	svcIP := result.Addr

	var announceNodes []string
	if result.Protocol == constant.PorterProtocolLayer2 {
		if len(nodesIP) == 0 {
			return nil, result.Sp.DelBalancer(svcIP)
		}

		index := rand.Int() % len(nodesIP)
		found := false
		preNodeIP, ok := svc.Annotations[constant.PorterLayer2Annotation]
		if ok {
			for i, nodeIP := range nodesIP {
				if nodeIP == preNodeIP {
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
			svc.Annotations[constant.PorterLayer2Annotation] = nodesIP[index]

			err = r.Update(context.Background(), svc)
			if err != nil {
				return nil, err
			}
		}

		announceNodes = append(announceNodes, nodesIP[index])
	} else {
		announceNodes = append(announceNodes, nodesIP...)
	}

	err = result.Sp.SetBalancer(svcIP, announceNodes)
	ctrl.Log.Info("callSetLoadBalancer", "result", result,
		"announceNodes", announceNodes, "avaliableNodes", nodesIP, "err", err)
	return announceNodes, err

}

func (r *ServiceReconciler) callDelLoadBalancer(result ipam.IPAMResult, svc corev1.Service) error {
	if len(svc.Status.LoadBalancer.Ingress) <= 0 {
		return nil
	}

	ip := svc.Status.LoadBalancer.Ingress[0].IP

	err := result.Sp.DelBalancer(ip)
	ctrl.Log.Info("callDelLoadBalancer", "result", result,
		"IngressIP", ip, "err", err)
	return err
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

	if util.IsDeletionCandidate(svc, constant.FinalizerName) {
		if result.ShouldUnAssignIP() {
			err = r.callDelLoadBalancer(result, *svc)
			if err != nil {
				r.Event(svc, corev1.EventTypeWarning, ReasonDeleteLoadBalancer, fmt.Sprintf(DelLoadBalancerFailedMsg, err))
				return ctrl.Result{}, err
			}
		}

		if result.ShouldUnAssignIP() {
			_, err = ipam.IPAMAllocator.UnAssignIP(args, false)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		controllerutil.RemoveFinalizer(svc, constant.FinalizerName)
		err = r.Update(context.Background(), svc)
		log.Info("RemoveFinalizer", "finalizer", svc.Finalizers, "err", err)
		return ctrl.Result{}, err
	}

	if util.NeedToAddFinalizer(svc, constant.FinalizerName) {
		controllerutil.AddFinalizer(svc, constant.FinalizerName)
		err := r.Update(context.Background(), svc)
		log.Info("AddFinalizer", "finalizer", svc.Finalizers, "err", err)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	clone := svc.DeepCopy()
	clone.Status.LoadBalancer.Ingress = nil

	// Check if the IP address specified by the service should be changed.
	if args.ShouldUnAssignIP(result) {
		err = r.callDelLoadBalancer(result, *svc)
		if err != nil {
			r.Event(svc, corev1.EventTypeWarning, ReasonDeleteLoadBalancer, fmt.Sprintf(DelLoadBalancerFailedMsg, err))
			return ctrl.Result{}, err
		}

		r.Event(svc, corev1.EventTypeNormal, ReasonDeleteLoadBalancer, fmt.Sprintf(DelLoadBalancerMsg, args.Addr, result.Addr))
		_, err = ipam.IPAMAllocator.UnAssignIP(args, false)
		if err != nil {
			return ctrl.Result{}, err
		}

		result.Clean()
	}

	if result.ShouldAssignIP() {
		result, err = ipam.IPAMAllocator.AssignIP(args)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	clone.Status.LoadBalancer.Ingress = append(clone.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{
		IP: result.Addr,
	})

	nodes, err := r.callSetLoadBalancer(result, clone)
	if err != nil {
		r.Event(svc, corev1.EventTypeWarning, ReasonAddLoadBalancer, fmt.Sprintf(AddLoadBalancerFailedMsg, nodes, err))
		return ctrl.Result{}, err
	}

	r.Event(svc, corev1.EventTypeNormal, ReasonAddLoadBalancer, fmt.Sprintf(AddLoadBalancerMsg, nodes))
	if reflect.DeepEqual(svc, clone) {
		return ctrl.Result{}, nil
	}

	err = r.Status().Update(context.Background(), clone)
	log.Info("UpdateIngress", "Ingress", clone.Status.LoadBalancer.Ingress, "err", err)
	return ctrl.Result{}, err
}

func (r *ServiceReconciler) constructIPAMArgs(svc *corev1.Service) ipam.IPAMArgs {
	args := ipam.IPAMArgs{}

	args.Key = types.NamespacedName{
		Name:      svc.Name,
		Namespace: svc.Namespace,
	}.String()

	if svc.Annotations != nil {
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
func (r *ServiceReconciler) getServiceNodes(svc *corev1.Service) ([]string, error) {
	endpoints := &corev1.Endpoints{}

	err := r.Get(context.TODO(), types.NamespacedName{Namespace: svc.GetNamespace(), Name: svc.GetName()}, endpoints)
	if err != nil {
		return nil, err
	}

	active := make([]string, 0)
	for _, subnet := range endpoints.Subsets {
		for _, addr := range subnet.Addresses {
			active = append(active, *addr.NodeName)
		}
	}
	if len(active) <= 0 {
		return nil, nil
	}

	nodeIPs, err := r.getNodeIPs()
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	if svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal {
		set.FromArray(active).Iter(func(item interface{}) error {
			result = append(result, nodeIPs[item.(string)])
			return nil
		})
	} else {
		for _, nodeIP := range nodeIPs {
			result = append(result, nodeIP)
		}
	}

	return result, nil
}

func (r *ServiceReconciler) getNodeIPs() (map[string]string, error) {
	nodeList := &corev1.NodeList{}

	err := r.List(context.TODO(), nodeList)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, node := range nodeList.Items {
		ip := util.GetNodeIP(node)
		if ip != nil {
			result[node.Name] = ip.String()
		}
	}
	return result, nil
}

func SetupServiceReconciler(mgr ctrl.Manager) error {
	lb := &ServiceReconciler{
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor("PorterLB Manager"),
	}
	err := lb.SetupWithManager(mgr)
	return err
}
