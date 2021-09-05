package lb

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/kubesphere/porterlb/pkg/constant"
	"github.com/kubesphere/porterlb/pkg/util"
	"github.com/kubesphere/porterlb/pkg/validate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func isProxyResc(obj metav1.Object) bool {
	return obj.GetNamespace() == envNamespace() && strings.HasPrefix(obj.GetName(), constant.PorterDeDsPrefix)
}

func envNamespace() string {
	ns := os.Getenv(constant.EnvPorterNamespace)
	if ns == "" {
		return constant.PorterNamespace
	}
	return ns
}

// eg. PorterLB LBS name/namespace: `nginx`/`default`, DaemonSet/Deployment/Pod name: `svc-proxy-nginx-default`
func proxyRescName(svcName, svcNs string) string {
	return constant.PorterDeDsPrefix + svcName + constant.NameSeparator + svcNs
}

func svcName(rescName, svcNs string) string {
	return rescName[len(constant.PorterDeDsPrefix) : len(rescName)-len(svcNs)-len(constant.NameSeparator)]
}

func (r *ServiceReconciler) shouldReconcileDeDs(e metav1.Object) bool {
	if !isProxyResc(e) {
		return false
	}
	svc := &corev1.Service{}
	ns, ok := e.GetAnnotations()[constant.PorterLBSAnnotationKey]
	if !ok {
		return false
	}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: svcName(e.GetName(), ns)}, svc)
	if err != nil {
		return !errors.IsNotFound(err)
	}

	return IsPorterLBService(svc)
}

func newProxyResc(svc *corev1.Service) *runtime.Object {
	var proxyResc runtime.Object
	switch svc.Annotations[constant.PorterLBSAnnotationKey] {
	case constant.PorterOnePort:
		proxyResc = newProxyDe(svc)
	case constant.PorterAllNode:
		proxyResc = newProxyDs(svc)
	}
	return &proxyResc
}

func newProxyDe(svc *corev1.Service) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: *newProxyRescOM(svc),
		Spec: appsv1.DeploymentSpec{
			Selector: newProxyRescSel(svc),
			Template: *newProxyPoTepl(svc),
		},
	}
}

func newProxyDs(svc *corev1.Service) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: *newProxyRescOM(svc),
		Spec: appsv1.DaemonSetSpec{
			Selector: newProxyRescSel(svc),
			Template: *newProxyPoTepl(svc),
		},
	}
}

func newProxyRescAnno(svc *corev1.Service) *map[string]string {
	return &map[string]string{
		constant.PorterAnnotationKey:    constant.PorterAnnotationValue,
		constant.PorterLBSAnnotationKey: svc.Namespace,
	}
}

func newProxyRescOM(svc *corev1.Service) *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Name:        proxyRescName(svc.Name, svc.Namespace),
		Namespace:   envNamespace(),
		Annotations: *newProxyRescAnno(svc),
	}
}

func newProxyRescSel(svc *corev1.Service) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"name": proxyRescName(svc.Name, svc.Namespace),
		},
	}
}

func newProxyPoTepl(svc *corev1.Service) *corev1.PodTemplateSpec {
	res := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"name": proxyRescName(svc.Name, svc.Namespace)},
		},
		Spec: corev1.PodSpec{
			Containers:     []corev1.Container{},
			InitContainers: []corev1.Container{*newForwardCtn(svc.Name)},
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{{
							MatchExpressions: []corev1.NodeSelectorRequirement{{
								Key:      constant.PorterNodeExcludeLBSLabel,
								Operator: corev1.NodeSelectorOpDoesNotExist,
							}},
						}},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
						{
							Weight: 10,
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      constant.PorterNodeExtnlIPPrefLabel,
									Operator: corev1.NodeSelectorOpExists,
								}},
							},
						},
					},
				},
			},
			// Make proxy pods runnable on master nodes
			Tolerations: []corev1.Toleration{{
				Key:      constant.KubernetesMasterLabel,
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			}},
		},
	}
	for index, svcPort := range svc.Spec.Ports {
		res.Spec.Containers = append(res.Spec.Containers, *newProxyCtn(
			proxyRescName(svc.Name, svc.Namespace)+strconv.Itoa(index),
			svc.Spec.ClusterIP,
			svcPort.Port,
			svcPort.Protocol,
		))
	}
	return res
}

func newProxyCtn(name, clusterIP string, port int32, proto corev1.Protocol) *corev1.Container {
	return &corev1.Container{
		Name:  name,
		Image: constant.PorterProxyImage,
		Ports: []corev1.ContainerPort{{
			HostPort:      port,
			ContainerPort: port,
			Protocol:      proto,
		}},
		Env: []corev1.EnvVar{
			{Name: "SVC_IP", Value: clusterIP},
			{Name: "SVC_PORT", Value: strconv.Itoa(int(port))},
			{Name: "POD_PORT", Value: strconv.Itoa(int(port))},
			{Name: "PROTO", Value: string(proto)},
		},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
		},
	}
}

func newForwardCtn(name string) *corev1.Container {
	privileged := true
	return &corev1.Container{
		Name:  name,
		Image: constant.PorterForwardImage,
		SecurityContext: &corev1.SecurityContext{
			Privileged: &privileged,
		},
	}
}

// Main procedure for ProterLB LBS
func (r *ServiceReconciler) reconcileLBSNormal(svc *corev1.Service) (ctrl.Result, error) {
	log := ctrl.Log.WithValues("service", types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name})
	log.Info("lbs reconciling")
	var err error

	if !util.ContainsString(svc.GetFinalizers(), constant.PorterLBSFInalizer) {
		controllerutil.AddFinalizer(svc, constant.PorterLBSFInalizer)
		if err := r.Update(context.Background(), svc); err != nil {
			log.Error(err, "can't register finalizer")
			return ctrl.Result{}, err
		}
	}

	// Check if all nodes are labeled for having external-ip
	// Labeled nodes are prefered for PorterlLB to deploy to
	nodeList := &corev1.NodeList{}
	if err = r.List(context.TODO(), nodeList); err != nil {
		log.Error(err, "can't get node information")
		return ctrl.Result{}, err
	}
	// use map for efficiency
	nodeWithExternalIP := map[string]*corev1.Node{}
	nodeWithInternalIP := map[string]*corev1.Node{}
	for _, node := range nodeList.Items {
		for _, nodeAddr := range node.Status.Addresses {
			if nodeAddr.Type == corev1.NodeExternalIP {
				nodeWithExternalIP[nodeAddr.Address] = &node
				if _, ok := node.Labels[constant.PorterNodeExtnlIPPrefLabel]; !ok {
					node.Labels[constant.PorterNodeExtnlIPPrefLabel] = ""
					if err = r.Update(context.Background(), &node); err != nil {
						log.Error(err, "can't label node")
						return ctrl.Result{}, err
					}
				}
			}
			if nodeAddr.Type == corev1.NodeInternalIP {
				nodeWithInternalIP[nodeAddr.Address] = &node
			}
		}
	}

	// Delete another kind of Proxy resource if exists
	// Passes when resource not exist or successfully deleted
	dpDsNamespacedName := types.NamespacedName{Namespace: envNamespace(), Name: proxyRescName(svc.Name, svc.Namespace)}
	var proxyResc, shouldRmResc runtime.Object
	switch svc.Annotations[constant.PorterLBSAnnotationKey] {
	case constant.PorterOnePort:
		shouldRmResc = &appsv1.DaemonSet{}
	case constant.PorterAllNode:
		shouldRmResc = &appsv1.Deployment{}
	default:
		log.Info("unsupport PorterLB LBS annotation value:" + svc.Annotations[constant.PorterLBSAnnotationKey])
		return ctrl.Result{}, nil
	}

	if err = r.Get(context.TODO(), dpDsNamespacedName, shouldRmResc); err == nil {
		if err = r.Delete(context.Background(), shouldRmResc); err != nil {
			log.Error(err, "can't remove another kind of proxy resource")
			return ctrl.Result{}, err
		}
	} else {
		if !errors.IsNotFound(err) {
			log.Error(err, "can't get another kind of proxy resource")
			return ctrl.Result{}, err
		}
	}

	// Check if specified Proxy resource exists
	switch svc.Annotations[constant.PorterLBSAnnotationKey] {
	case constant.PorterOnePort:
		proxyResc = &appsv1.Deployment{}
	case constant.PorterAllNode:
		proxyResc = &appsv1.DaemonSet{}
	}
	if err = r.Get(context.TODO(), dpDsNamespacedName, proxyResc); err == nil {
		// If exists
		// Update Service pod template by svc
		proxyResc = *newProxyResc(svc)
		if err = r.Update(context.Background(), proxyResc); err != nil {
			log.Error(err, "can't patch proxy resc")
			return ctrl.Result{}, err
		}
		// External-ip updating procedure
		podList := &corev1.PodList{}
		opts := []client.ListOption{
			client.InNamespace(envNamespace()),
			client.MatchingLabels{"name": proxyRescName(svc.Name, svc.Namespace)},
			client.MatchingFields{"status.phase": "Running"},
		}
		if err = r.List(context.TODO(), podList, opts...); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err, "can't list proxy pod")
				return ctrl.Result{}, err
			}
		}
		if len(podList.Items) == 0 {
			log.Info("no proxy pod available")
			return ctrl.Result{}, err
		}
		// Find suitable proxy pods which in nodes with external-ip or internal-ip
		podInExternalIPNode := []string{}
		podInInternalIPNode := []string{}
		for _, pod := range podList.Items {
			if _, ok := nodeWithExternalIP[pod.Status.HostIP]; ok {
				podInExternalIPNode = append(podInExternalIPNode, pod.Status.HostIP)
			}
			if _, ok := nodeWithInternalIP[pod.Status.HostIP]; ok {
				podInInternalIPNode = append(podInInternalIPNode, pod.Status.HostIP)
			}
		}
		// If there's some proxy pods in nodes with external-ip, use these nodes' external-ips as svc external-ip
		// Otherwize svc external-ip was specified by node internal-ips which there's a proxy pod in
		if len(podInExternalIPNode) != 0 {
			svc.ObjectMeta.Annotations[constant.PorterLBSExposedExternalIP] = strings.Join(podInExternalIPNode, constant.IPSeparator)
		} else {
			delete(svc.ObjectMeta.Annotations, constant.PorterLBSExposedExternalIP)
		}
		if len(podInInternalIPNode) != 0 {
			svc.ObjectMeta.Annotations[constant.PorterLBSExposedInternalIP] = strings.Join(podInInternalIPNode, constant.IPSeparator)
		} else {
			delete(svc.ObjectMeta.Annotations, constant.PorterLBSExposedInternalIP)
		}

		if err = r.Update(context.Background(), svc); err != nil {
			log.Error(err, "can't update svc exposed ips annotations")
			return ctrl.Result{}, err
		}
	} else {
		if !errors.IsNotFound(err) {
			log.Error(err, "can't get proxy resource")
			return ctrl.Result{}, err
		}
		// If not exists, create Proxy resource
		if err = r.Create(context.TODO(), *newProxyResc(svc)); err != nil {
			log.Error(err, "can't create proxy resource")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, err
}

// Called when PorterLB LBS Service was deleted
func (r *ServiceReconciler) reconcileLBSDelete(svc *corev1.Service) (ctrl.Result, error) {
	log := ctrl.Log.WithValues("service", types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name})
	log.Info("lbs reconciling deletion finalizing")
	var err error

	if util.ContainsString(svc.GetFinalizers(), constant.PorterLBSFInalizer) {
		dpDsNamespacedName := types.NamespacedName{Namespace: envNamespace(), Name: proxyRescName(svc.Name, svc.Namespace)}
		var proxyResc runtime.Object
		switch svc.Annotations[constant.PorterLBSAnnotationKey] {
		case constant.PorterOnePort:
			proxyResc = &appsv1.Deployment{}
		case constant.PorterAllNode:
			proxyResc = &appsv1.DaemonSet{}
		default:
			log.Info("unsupport PorterLB LBS annotation value:" + svc.Annotations[constant.PorterLBSAnnotationKey])
			return ctrl.Result{}, nil
		}
		err = r.Get(context.TODO(), dpDsNamespacedName, proxyResc)
		if err != nil {
			if errors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			log.Error(err, "can't get deployment/daemonset for deletion")
			return ctrl.Result{}, err
		}
		if err = r.Delete(context.Background(), proxyResc); err != nil {
			log.Error(err, "can't remove deployment/daemonset")
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(svc, constant.PorterLBSFInalizer)
		if err = r.Update(context.Background(), svc); err != nil {
			log.Error(err, "can't remove finalizer")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) reconcileLBS(svc *corev1.Service) (ctrl.Result, error) {
	if svc.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileLBSNormal(svc)
	}
	return r.reconcileLBSDelete(svc)
}

// Judge whether this load balancer should be exposed by Porter LB Service
// Such Service will be exposed by Proxy Pod
func IsPorterLBService(obj runtime.Object) bool {
	if svc, ok := obj.(*corev1.Service); ok {
		return validate.HasPorterLBAnnotation(svc.Annotations) && validate.IsTypeLoadBalancer(svc) && validate.HasPorterLBSAnnotation(svc.Annotations)
	}
	return false
}
