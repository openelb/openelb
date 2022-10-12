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
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/openelb/openelb/api/v1alpha2"
	networkv1alpha2 "github.com/openelb/openelb/api/v1alpha2"
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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	log logr.Logger
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
			return r.shouldReconcileDeDs(e.MetaNew) || r.shouldReconcileDeDs(e.MetaOld)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return r.shouldReconcileDeDs(e.Meta)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.shouldReconcileDeDs(e.Meta)
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
	return mgr.GetFieldIndexer().IndexField(context.TODO(), &corev1.Pod{}, "status.phase", func(rawObj runtime.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{string(pod.Status.Phase)}
	})
}

func (r *ServiceReconciler) callSetLoadBalancer(result ipam.IPAMResult, svc *corev1.Service) error {
	nodes, err := r.getServiceNodes(svc)
	if err != nil {
		return err
	}

	svcIP := result.Addr

	var announceNodes []corev1.Node
	if result.Protocol == constant.OpenELBProtocolLayer2 {
		if len(nodes) == 0 {
			return result.Sp.DelBalancer(svcIP)
		}

		index := rand.Int() % len(nodes)
		found := false
		preNode, ok := svc.Annotations[constant.OpenELBLayer2Annotation]
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
			svc.Annotations[constant.OpenELBLayer2Annotation] = nodes[index].Name

			err = r.Update(context.Background(), svc)
			if err != nil {
				return err
			}
		}

		announceNodes = append(announceNodes, nodes[index])
	} else {
		announceNodes = append(announceNodes, nodes...)
	}
	if result.Protocol == constant.OpenELBProtocolVip {
		vip := fmt.Sprintf("%s:%s", svcIP, svc.Namespace+"/"+svc.Name)
		return result.Sp.SetBalancer(vip, nil)
	}
	return result.Sp.SetBalancer(svcIP, announceNodes)
}

func (r *ServiceReconciler) callDelLoadBalancer(result ipam.IPAMResult, svc *corev1.Service) error {
	if result.Addr != "" {
		if svc.Annotations != nil && svc.Annotations[constant.OpenELBLayer2Annotation] != "" {
			delete(svc.Annotations, constant.OpenELBLayer2Annotation)
			err := r.Update(context.Background(), svc)
			if err != nil {
				return err
			}
		}
		if result.Protocol == constant.OpenELBProtocolVip {
			vip := fmt.Sprintf("%s:%s", result.Addr, svc.Namespace+"/"+svc.Name)
			return result.Sp.DelBalancer(vip)
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
	log.Info("setup openelb service")

	svc := &corev1.Service{}
	err := r.Get(context.TODO(), req.NamespacedName, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Reconcile by OpenELB NodeProxy if this service is specified to be exported by it
	if validate.HasOpenELBNPAnnotation(svc.Annotations) {
		return r.reconcileNP(svc)
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
		clone.Labels[constant.OpenELBEIPAnnotationKeyV1Alpha2] = result.Eip
	} else {
		controllerutil.RemoveFinalizer(clone, constant.FinalizerName)
		delete(clone.Labels, constant.OpenELBEIPAnnotationKeyV1Alpha2)
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
		if _, ok := svc.Annotations[constant.OpenELBAnnotationKey]; ok &&
			svc.Spec.Type == corev1.ServiceTypeLoadBalancer &&
			svc.DeletionTimestamp == nil {
			args.Unalloc = false
		}

		if ip, ok := svc.Annotations[constant.OpenELBEIPAnnotationKey]; ok {
			args.Addr = ip
		}

		if eip, ok := svc.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2]; ok {
			args.Eip = eip
		}

		if protocol, ok := svc.Annotations[constant.OpenELBProtocolAnnotationKey]; ok {
			args.Protocol = protocol
		} else {
			args.Protocol = constant.OpenELBProtocolBGP
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
			if addr.NodeName == nil {
				continue
			}
			active[*addr.NodeName] = true
		}
	}

	//2. get next hops
	nodeList := &corev1.NodeList{}
	err = r.List(context.TODO(), nodeList)
	if err != nil {
		return nil, err
	}

	resultNodes := make([]corev1.Node, 0)
	if svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal && len(active) > 0 {
		for _, node := range nodeList.Items {
			if active[node.Name] {
				resultNodes = append(resultNodes, node)
			}
		}

		return resultNodes, nil
	}

	if svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal && len(active) == 0 {
		clone := svc.DeepCopy()
		clone.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeCluster
		_ = r.Update(context.Background(), clone)
		r.log.Info(fmt.Sprintf("endpoint don't have nodeName, so cannot set externalTrafficPolicy to Local"))
	}

	for _, node := range nodeList.Items {
		if nodeReady(&node) {
			resultNodes = append(resultNodes, node)
		}
	}

	return resultNodes, nil
}

func SetupServiceReconciler(mgr ctrl.Manager) error {
	lb := &ServiceReconciler{
		Client:        mgr.GetClient(),
		log:           ctrl.Log.WithName("Manager"),
		EventRecorder: mgr.GetEventRecorderFor("Manager"),
	}
	err := lb.SetupWithManager(mgr)
	return err
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

// +kubebuilder:webhook:path=/validate-network-kubesphere-io-v1alpha2-svc,mutating=true,sideEffects=NoneOnDryRun,failurePolicy=fail,groups="",resources=services,verbs=create,versions=v1,name=mutating.eip.network.kubesphere.io

type SvcAnnotator struct {
	client.Client
	decoder *admission.Decoder
}

func (r *SvcAnnotator) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}

func (r *SvcAnnotator) Handle(ctx context.Context, req admission.Request) admission.Response {
	svc := &corev1.Service{}

	if err := r.decoder.Decode(req, svc); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		marshaledSvc, err := json.Marshal(svc)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		return admission.PatchResponseFromRaw(req.Object.Raw, marshaledSvc)
	}
	// check default eip
	eips := networkv1alpha2.EipList{}
	err := r.List(context.Background(), &eips)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	for _, eip := range eips.Items {
		if validate.HasOpenELBDefaultEipAnnotation(eip.Annotations) {
			// exist default eip,injection annotation
			if svc.Annotations == nil {
				svc.Annotations = make(map[string]string)
				svc.Annotations[constant.OpenELBAnnotationKey] = constant.OpenELBAnnotationValue
			} else if value, ok := svc.Annotations[constant.OpenELBAnnotationKey]; !ok || value != constant.OpenELBAnnotationValue {
				svc.Annotations[constant.OpenELBAnnotationKey] = constant.OpenELBAnnotationValue
			}
			if _, ok := svc.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2]; !ok {
				svc.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2] = eip.Name
				svc.Annotations[constant.OpenELBProtocolAnnotationKey] = eip.GetProtocol()
			}
			break
		}
	}
	marshaledSvc, err := json.Marshal(svc)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledSvc)
}
