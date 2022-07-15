package lb

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/util"
	"github.com/openelb/openelb/pkg/validate"
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
	return obj.GetNamespace() == util.EnvNamespace() && strings.HasPrefix(obj.GetName(), constant.NodeProxyWorkloadPrefix)
}

// eg. OpenELB NodeProxy name/namespace: `nginx`/`default`, DaemonSet/Deployment/Pod name: `svc-proxy-nginx-default`
func proxyRescName(svcName, svcNs string) string {
	return constant.NodeProxyWorkloadPrefix + svcName + constant.NameSeparator + svcNs
}

func svcName(rescName, svcNs string) string {
	return rescName[len(constant.NodeProxyWorkloadPrefix) : len(rescName)-len(svcNs)-len(constant.NameSeparator)]
}

func (r *ServiceReconciler) shouldReconcileDeDs(e metav1.Object) bool {
	if !isProxyResc(e) {
		return false
	}
	svc := &corev1.Service{}
	ns, ok := e.GetAnnotations()[constant.NodeProxyTypeAnnotationKey]
	if !ok {
		return false
	}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: svcName(e.GetName(), ns)}, svc)
	if err != nil {
		return !errors.IsNotFound(err)
	}

	return IsOpenELBNPService(svc)
}

func (r *ServiceReconciler) newProxyResc(svc *corev1.Service) *runtime.Object {
	var proxyResc runtime.Object
	switch svc.Annotations[constant.NodeProxyTypeAnnotationKey] {
	case constant.NodeProxyTypeDeployment:
		proxyResc = r.newProxyDe(svc)
	case constant.NodeProxyTypeDaemonSet:
		proxyResc = r.newProxyDs(svc)
	}
	return &proxyResc
}

func (r *ServiceReconciler) newProxyDe(svc *corev1.Service) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: *r.newProxyRescOM(svc),
		Spec: appsv1.DeploymentSpec{
			Selector: r.newProxyRescSel(svc),
			Template: *r.newProxyPoTepl(svc),
		},
	}
}

func (r *ServiceReconciler) newProxyDs(svc *corev1.Service) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: *r.newProxyRescOM(svc),
		Spec: appsv1.DaemonSetSpec{
			Selector: r.newProxyRescSel(svc),
			Template: *r.newProxyPoTepl(svc),
		},
	}
}

func (r *ServiceReconciler) newProxyRescAnno(svc *corev1.Service) *map[string]string {
	return &map[string]string{
		constant.OpenELBAnnotationKey:       constant.OpenELBAnnotationValue,
		constant.NodeProxyTypeAnnotationKey: svc.Namespace,
	}
}

func (r *ServiceReconciler) newProxyRescOM(svc *corev1.Service) *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Name:        proxyRescName(svc.Name, svc.Namespace),
		Namespace:   util.EnvNamespace(),
		Annotations: *r.newProxyRescAnno(svc),
	}
}

func (r *ServiceReconciler) newProxyRescSel(svc *corev1.Service) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"name": proxyRescName(svc.Name, svc.Namespace),
		},
	}
}

func (r *ServiceReconciler) newProxyPoTepl(svc *corev1.Service) *corev1.PodTemplateSpec {
	res := &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"name": proxyRescName(svc.Name, svc.Namespace)},
		},
		Spec: corev1.PodSpec{
			Containers:     []corev1.Container{*r.newProxyCtn(svc)},
			InitContainers: []corev1.Container{*r.newForwardCtn(svc.Name)},
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{{
							MatchExpressions: []corev1.NodeSelectorRequirement{{
								Key:      constant.LabelNodeProxyExcludeNode,
								Operator: corev1.NodeSelectorOpDoesNotExist,
							}},
						}},
					},
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
						{
							Weight: 10,
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      constant.LabelNodeProxyExternalIPPreffered,
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
	return res
}

// User can config NodeProxy by ConfigMap to specify the proxy and forward images
// If the ConfigMap exists and the configuration is set, use it,
// 	otherwise, use the default image got from constants.
func (r *ServiceReconciler) getNPConfig() (*corev1.ConfigMap, error) {
	NPCfgName := types.NamespacedName{Namespace: util.EnvNamespace(), Name: constant.OpenELBImagesConfigMap}
	cm := &corev1.ConfigMap{}
	err := r.Get(context.Background(), NPCfgName, cm)
	return cm, err
}

func (r *ServiceReconciler) getForwardImage() string {
	cm, err := r.getNPConfig()
	if err != nil {
		return constant.NodeProxyDefaultForwardImage
	}

	image, exist := cm.Data[constant.NodeProxyConfigMapForwardImage]
	if !exist {
		return constant.NodeProxyDefaultForwardImage
	}

	return image
}

func (r *ServiceReconciler) getProxyImage() string {
	cm, err := r.getNPConfig()
	if err != nil {
		return constant.NodeProxyDefaultProxyImage
	}

	image, exist := cm.Data[constant.NodeProxyConfigMapProxyImage]
	if !exist {
		return constant.NodeProxyDefaultProxyImage
	}

	return image
}

// The only env variable is `PROXY_ARGS`
// `PROXY_ARGS` is 4-tuple parameters split by space: <SVC_IP POD_PORT SVC_PORT SVC_PROTO>
func (r *ServiceReconciler) newProxyCtnEnvArgs(ports *[]corev1.ServicePort, clusterIP string) *[]corev1.EnvVar {
	var builder strings.Builder
	for _, port := range *ports {
		builder.WriteString(clusterIP)
		builder.WriteString(constant.EnvArgSplitter)
		builder.WriteString(strconv.Itoa(int(port.Port)))
		builder.WriteString(constant.EnvArgSplitter)
		builder.WriteString(strconv.Itoa(int(port.Port)))
		builder.WriteString(constant.EnvArgSplitter)
		builder.WriteString(strings.ToLower(string(port.Protocol)))
		builder.WriteString(constant.EnvArgSplitter)
	}
	return &[]corev1.EnvVar{{
		Name:  "PROXY_ARGS",
		Value: builder.String(),
	}}
}

func (r *ServiceReconciler) newProxyCtnPorts(ports *[]corev1.ServicePort) *[]corev1.ContainerPort {
	res := make([]corev1.ContainerPort, len(*ports))
	for i, port := range *ports {
		res[i].ContainerPort = port.Port
		res[i].HostPort = port.Port
		res[i].Name = port.Name
		res[i].Protocol = port.Protocol
	}
	return &res
}

func (r *ServiceReconciler) newProxyCtn(svc *corev1.Service) *corev1.Container {
	return &corev1.Container{
		Name:  proxyRescName(svc.Name, svc.Namespace),
		Image: r.getProxyImage(),
		Ports: *r.newProxyCtnPorts(&svc.Spec.Ports),
		Env:   *r.newProxyCtnEnvArgs(&svc.Spec.Ports, svc.Spec.ClusterIP),
		// NET_ADMIN capability is required for iptables running in a container
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
		},
	}
}

func (r *ServiceReconciler) newForwardCtn(name string) *corev1.Container {
	privileged := true
	return &corev1.Container{
		Name:  name,
		Image: r.getForwardImage(),
		SecurityContext: &corev1.SecurityContext{
			Privileged: &privileged,
		},
	}
}

// Main procedure for ProterLB NodeProxy
func (r *ServiceReconciler) reconcileNPNormal(svc *corev1.Service) (ctrl.Result, error) {
	log := ctrl.Log.WithValues("service", types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name})
	log.Info("node-proxy reconciling")
	var err error

	if !util.ContainsString(svc.GetFinalizers(), constant.NodeProxyFinalizerName) {
		controllerutil.AddFinalizer(svc, constant.NodeProxyFinalizerName)
		if err := r.Update(context.Background(), svc); err != nil {
			log.Error(err, "can't register finalizer")
			return ctrl.Result{}, err
		}
	}

	// Check if all nodes are labeled for having external-ip
	// Labeled nodes are prefered for OpenELB to deploy to
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
				if _, ok := node.Labels[constant.LabelNodeProxyExternalIPPreffered]; !ok {
					node.Labels[constant.LabelNodeProxyExternalIPPreffered] = ""
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
	dpDsNamespacedName := types.NamespacedName{Namespace: util.EnvNamespace(), Name: proxyRescName(svc.Name, svc.Namespace)}
	var proxyResc, shouldRmResc runtime.Object
	switch svc.Annotations[constant.NodeProxyTypeAnnotationKey] {
	case constant.NodeProxyTypeDeployment:
		shouldRmResc = &appsv1.DaemonSet{}
	case constant.NodeProxyTypeDaemonSet:
		shouldRmResc = &appsv1.Deployment{}
	default:
		log.Info("unsupport OpenELB NodeProxy annotation value:" + svc.Annotations[constant.NodeProxyTypeAnnotationKey])
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
	switch svc.Annotations[constant.NodeProxyTypeAnnotationKey] {
	case constant.NodeProxyTypeDeployment:
		proxyResc = &appsv1.Deployment{}
	case constant.NodeProxyTypeDaemonSet:
		proxyResc = &appsv1.DaemonSet{}
	}
	if err = r.Get(context.TODO(), dpDsNamespacedName, proxyResc); err == nil {
		// If exists
		// Update Service pod template by svc
		proxyResc = *r.newProxyResc(svc)
		if err = r.Update(context.Background(), proxyResc); err != nil {
			log.Error(err, "can't patch proxy resc")
			return ctrl.Result{}, err
		}
		// External-ip updating procedure
		podList := &corev1.PodList{}
		opts := []client.ListOption{
			client.InNamespace(util.EnvNamespace()),
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
			sort.Strings(podInExternalIPNode) // Use sorting to guarantee update idempotency
			svc.ObjectMeta.Annotations[constant.NodeProxyExternalIPAnnotationKey] = strings.Join(podInExternalIPNode, constant.IPSeparator)
		} else {
			delete(svc.ObjectMeta.Annotations, constant.NodeProxyExternalIPAnnotationKey)
		}
		if len(podInInternalIPNode) != 0 {
			sort.Strings(podInInternalIPNode) // Use sorting to guarantee update idempotency
			svc.ObjectMeta.Annotations[constant.NodeProxyInternalIPAnnotationKey] = strings.Join(podInInternalIPNode, constant.IPSeparator)
		} else {
			delete(svc.ObjectMeta.Annotations, constant.NodeProxyInternalIPAnnotationKey)
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
		if err = r.Create(context.TODO(), *r.newProxyResc(svc)); err != nil {
			log.Error(err, "can't create proxy resource")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, err
}

// Called when OpenELB NodeProxy Service was deleted
func (r *ServiceReconciler) reconcileNPDelete(svc *corev1.Service) (ctrl.Result, error) {
	log := ctrl.Log.WithValues("service", types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name})
	log.Info("node proxy reconciling deletion finalizing")
	var err error

	if util.ContainsString(svc.GetFinalizers(), constant.NodeProxyFinalizerName) {
		dpDsNamespacedName := types.NamespacedName{Namespace: util.EnvNamespace(), Name: proxyRescName(svc.Name, svc.Namespace)}
		var proxyResc runtime.Object
		switch svc.Annotations[constant.NodeProxyTypeAnnotationKey] {
		case constant.NodeProxyTypeDeployment:
			proxyResc = &appsv1.Deployment{}
		case constant.NodeProxyTypeDaemonSet:
			proxyResc = &appsv1.DaemonSet{}
		default:
			log.Info("unsupport OpenELB NodeProxy annotation value:" + svc.Annotations[constant.NodeProxyTypeAnnotationKey])
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
		controllerutil.RemoveFinalizer(svc, constant.NodeProxyFinalizerName)
		if err = r.Update(context.Background(), svc); err != nil {
			log.Error(err, "can't remove finalizer")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) reconcileNP(svc *corev1.Service) (ctrl.Result, error) {
	if svc.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileNPNormal(svc)
	}
	return r.reconcileNPDelete(svc)
}

// Judge whether this load balancer should be exposed by OpenELB Service
// Such Service will be exposed by Proxy Pod
func IsOpenELBNPService(obj runtime.Object) bool {
	if svc, ok := obj.(*corev1.Service); ok {
		return validate.HasOpenELBAnnotation(svc.Annotations) && validate.IsTypeLoadBalancer(svc) && validate.HasOpenELBNPAnnotation(svc.Annotations)
	}
	return false
}
