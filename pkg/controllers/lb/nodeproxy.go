package lb

import (
	"context"
	"reflect"
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
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
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
	ns, ok := e.GetAnnotations()[constant.NodeProxyNamespaceAnnotationKey]
	if !ok {
		return false
	}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: svcName(e.GetName(), ns)}, svc)
	if err != nil {
		return !errors.IsNotFound(err)
	}

	return IsOpenELBNPService(svc)
}

func (r *ServiceReconciler) newProxyResc(owner *metav1.ObjectMeta, svc *corev1.Service) client.Object {
	var proxyResc client.Object
	switch svc.Annotations[constant.NodeProxyTypeAnnotationKey] {
	case constant.NodeProxyTypeDeployment:
		proxyResc = r.newProxyDe(owner, svc)
	case constant.NodeProxyTypeDaemonSet:
		proxyResc = r.newProxyDs(owner, svc)
	}
	return proxyResc
}

func (r *ServiceReconciler) newProxyDe(owner *metav1.ObjectMeta, svc *corev1.Service) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: r.newProxyRescOM(owner, svc),
		Spec: appsv1.DeploymentSpec{
			Selector: r.newProxyRescSel(svc),
			Template: r.newProxyPoTepl(svc),
		},
	}
}

func (r *ServiceReconciler) newProxyDs(owner *metav1.ObjectMeta, svc *corev1.Service) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: r.newProxyRescOM(owner, svc),
		Spec: appsv1.DaemonSetSpec{
			Selector: r.newProxyRescSel(svc),
			Template: r.newProxyPoTepl(svc),
		},
	}
}

func (r *ServiceReconciler) newProxyRescAnno(svc *corev1.Service) map[string]string {
	return map[string]string{
		constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
		constant.NodeProxyNamespaceAnnotationKey: svc.Namespace,
	}
}

func (r *ServiceReconciler) newProxyRescOM(owner *metav1.ObjectMeta, svc *corev1.Service) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        proxyRescName(svc.Name, svc.Namespace),
		Namespace:   util.EnvNamespace(),
		Annotations: r.newProxyRescAnno(svc),
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion:         "apps/v1",
			Kind:               "Deployment",
			Name:               owner.GetName(),
			UID:                owner.GetUID(),
			BlockOwnerDeletion: ptr.To(true),
			Controller:         ptr.To(true),
		}},
	}
}

func (r *ServiceReconciler) newProxyRescSel(svc *corev1.Service) *metav1.LabelSelector {
	return &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"name": proxyRescName(svc.Name, svc.Namespace),
		},
	}
}

func (r *ServiceReconciler) newProxyPoTepl(svc *corev1.Service) corev1.PodTemplateSpec {
	res := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"name": proxyRescName(svc.Name, svc.Namespace)},
		},
		Spec: corev1.PodSpec{
			Containers:     []corev1.Container{r.newProxyCtn(svc)},
			InitContainers: []corev1.Container{r.newForwardCtn(svc.Name)},
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
			}, {
				Key:      constant.KubernetesControlPlaneLabel,
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			}},
		},
	}
	return res
}

// User can config NodeProxy by ConfigMap to specify the proxy and forward images
// If the ConfigMap exists and the configuration is set, use it,
//
//	otherwise, use the default image got from constants.
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
func (r *ServiceReconciler) newProxyCtnEnvArgs(ports *[]corev1.ServicePort, clusterIP string) []corev1.EnvVar {
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
	return []corev1.EnvVar{{
		Name:  "PROXY_ARGS",
		Value: builder.String(),
	}}
}

func (r *ServiceReconciler) newProxyCtnPorts(ports *[]corev1.ServicePort) []corev1.ContainerPort {
	res := make([]corev1.ContainerPort, len(*ports))
	for i, port := range *ports {
		res[i].ContainerPort = port.Port
		res[i].HostPort = port.Port
		res[i].Name = port.Name
		res[i].Protocol = port.Protocol
	}
	return res
}

func (r *ServiceReconciler) newProxyCtn(svc *corev1.Service) corev1.Container {
	return corev1.Container{
		Name:  proxyRescName(svc.Name, svc.Namespace),
		Image: r.getProxyImage(),
		Ports: r.newProxyCtnPorts(&svc.Spec.Ports),
		Env:   r.newProxyCtnEnvArgs(&svc.Spec.Ports, svc.Spec.ClusterIP),
		// NET_ADMIN capability is required for iptables running in a container
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
		},
		ImagePullPolicy:          corev1.PullIfNotPresent,
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
	}
}

func (r *ServiceReconciler) newForwardCtn(name string) corev1.Container {
	privileged := true
	return corev1.Container{
		Name:            name,
		Image:           r.getForwardImage(),
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{
			Privileged: &privileged,
		},
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
	}
}

// Main procedure for OpenELB NodeProxy
func (r *ServiceReconciler) reconcileNPNormal(ctx context.Context, svc *corev1.Service) (ctrl.Result, error) {
	klog.V(4).Infof("Starting to reconcile node-proxy %s/%s", svc.GetNamespace(), svc.GetName())

	if !util.ContainsString(svc.GetFinalizers(), constant.NodeProxyFinalizerName) {
		controllerutil.AddFinalizer(svc, constant.NodeProxyFinalizerName)
		if err := r.Update(ctx, svc); err != nil {
			klog.Errorf("can't register finalizer: %v", err)
			return ctrl.Result{}, err
		}
	}

	dpDsNamespacedName := types.NamespacedName{Namespace: util.EnvNamespace(), Name: proxyRescName(svc.Name, svc.Namespace)}
	nptypes := svc.Annotations[constant.NodeProxyTypeAnnotationKey]
	if nptypes != constant.NodeProxyTypeDeployment && nptypes != constant.NodeProxyTypeDaemonSet {
		klog.Info("unsupport OpenELB NodeProxy annotation value:" + nptypes)
		return ctrl.Result{}, nil
	}

	if err := r.isNodePorxyTypeChanged(ctx, dpDsNamespacedName, nptypes); err != nil {
		return ctrl.Result{}, err
	}

	// Check if specified Proxy resource exists
	var proxyResc client.Object
	switch nptypes {
	case constant.NodeProxyTypeDeployment:
		proxyResc = &appsv1.Deployment{}
	case constant.NodeProxyTypeDaemonSet:
		proxyResc = &appsv1.DaemonSet{}
	}
	if err := r.Get(ctx, dpDsNamespacedName, proxyResc); err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("can't get proxy resource: %v", err)
			return ctrl.Result{}, err
		}

		owner := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: util.EnvNamespace(), Name: util.EnvDeploymentName()}, owner); err != nil {
			return ctrl.Result{}, err
		}

		// If not exists, create Proxy resource
		obj := r.newProxyResc(&owner.ObjectMeta, svc)
		if err = r.Create(ctx, obj); err != nil && !errors.IsAlreadyExists(err) {
			klog.Errorf("can't create proxy resource: %v", err)
			return ctrl.Result{}, err
		}

		klog.Infof("create node-proxy %s/%s successfully", nptypes, obj.GetName())
		return ctrl.Result{}, nil
	}

	return r.updateNodeProxyResult(ctx, proxyResc, svc, nptypes)
}

// Delete another kind of Proxy resource if exists
// Passes when resource not exist or successfully deleted
func (r *ServiceReconciler) isNodePorxyTypeChanged(ctx context.Context, name types.NamespacedName, nptypes string) error {
	var shouldRmResc client.Object
	switch nptypes {
	case constant.NodeProxyTypeDeployment:
		shouldRmResc = &appsv1.DaemonSet{}
	case constant.NodeProxyTypeDaemonSet:
		shouldRmResc = &appsv1.Deployment{}
	}

	if err := r.Get(ctx, name, shouldRmResc); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}

		klog.Errorf("can't get another kind of proxy resource: %v", err)
		return err
	}

	if err := r.Delete(ctx, shouldRmResc); err != nil {
		klog.Errorf("can't remove another kind of proxy resource: %v", err)
		return err
	}

	klog.Infof("node-proxy type changed. delete %s/%s successfully", reflect.TypeOf(shouldRmResc).String(), shouldRmResc.GetName())
	return nil
}

func (r *ServiceReconciler) updateNodeProxyResult(ctx context.Context, proxyResc client.Object, svc *corev1.Service, nptypes string) (ctrl.Result, error) {
	// If exists - Update Service pod template by svc
	owner := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: util.EnvNamespace(), Name: util.EnvDeploymentName()}, owner); err != nil {
		return ctrl.Result{}, err
	}

	new := r.newProxyResc(&owner.ObjectMeta, svc)
	if r.needsUpdateNPWorkload(new, proxyResc, nptypes) {
		if err := r.Update(ctx, new); err != nil {
			klog.Errorf("can't patch proxy resc: %v", err)
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Check if all nodes are labeled for having external-ip
	// Labeled nodes are prefered for OpenELB to deploy to
	nodeList := &corev1.NodeList{}
	if err := r.List(ctx, nodeList); err != nil {
		klog.Errorf("can't get node information: %v", err)
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
					if err := r.Update(ctx, &node); err != nil {
						klog.Errorf("can't label node: %v", err)
						return ctrl.Result{}, err
					}
				}
			}
			if nodeAddr.Type == corev1.NodeInternalIP {
				nodeWithInternalIP[nodeAddr.Address] = &node
			}
		}
	}

	// External-ip updating procedure
	podList := &corev1.PodList{}
	opts := []client.ListOption{
		client.InNamespace(util.EnvNamespace()),
		client.MatchingLabels{"name": proxyRescName(svc.Name, svc.Namespace)},
		client.MatchingFields{"status.phase": "Running"},
	}
	if err := r.List(ctx, podList, opts...); err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("can't list proxy pod: %v", err)
			return ctrl.Result{}, err
		}
	}
	if len(podList.Items) == 0 {
		klog.V(4).Info("no proxy pod available")
		return ctrl.Result{}, nil
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
	clone := svc.DeepCopy()
	ips := []corev1.LoadBalancerIngress{}
	if len(podInExternalIPNode) != 0 {
		sort.Strings(podInExternalIPNode) // Use sorting to guarantee update idempotency
		clone.Annotations[constant.NodeProxyExternalIPAnnotationKey] = strings.Join(podInExternalIPNode, constant.IPSeparator)

		for _, ip := range podInExternalIPNode {
			ips = append(ips, corev1.LoadBalancerIngress{Hostname: "lb-" + ip})
		}
	} else {
		delete(clone.Annotations, constant.NodeProxyExternalIPAnnotationKey)
	}
	if len(podInInternalIPNode) != 0 {
		sort.Strings(podInInternalIPNode) // Use sorting to guarantee update idempotency
		clone.Annotations[constant.NodeProxyInternalIPAnnotationKey] = strings.Join(podInInternalIPNode, constant.IPSeparator)

		for _, ip := range podInInternalIPNode {
			ips = append(ips, corev1.LoadBalancerIngress{Hostname: "lb-" + ip})
		}
	} else {
		delete(clone.Annotations, constant.NodeProxyInternalIPAnnotationKey)
	}

	if !reflect.DeepEqual(svc.Annotations, clone.Annotations) {
		if err := r.Update(ctx, clone); err != nil {
			klog.Errorf("can't update svc exposed ips annotations: %v", err)
			return ctrl.Result{}, err
		}
	}

	clone.Status.LoadBalancer.Ingress = ips
	if !reflect.DeepEqual(svc.Status.LoadBalancer.Ingress, clone.Status.LoadBalancer.Ingress) {
		if err := r.Status().Update(ctx, clone); err != nil {
			klog.Errorf("can't update svc status: %v", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) needsUpdateNPWorkload(new client.Object, obj client.Object, nptypes string) bool {
	switch nptypes {
	case constant.NodeProxyTypeDeployment:
		objDeploy := obj.(*appsv1.Deployment)
		newDeploy := new.(*appsv1.Deployment)
		if reflect.DeepEqual(objDeploy.Spec.Selector, newDeploy.Spec.Selector) &&
			reflect.DeepEqual(objDeploy.Spec.Template.ObjectMeta, newDeploy.Spec.Template.ObjectMeta) &&
			reflect.DeepEqual(objDeploy.Spec.Template.Spec.Containers, newDeploy.Spec.Template.Spec.Containers) &&
			reflect.DeepEqual(objDeploy.Spec.Template.Spec.InitContainers, newDeploy.Spec.Template.Spec.InitContainers) &&
			reflect.DeepEqual(objDeploy.OwnerReferences, newDeploy.OwnerReferences) {
			for key, value := range newDeploy.Annotations {
				if objvalue, exist := objDeploy.Annotations[key]; !exist || objvalue != value {
					return true
				}
			}

			return false
		}

	case constant.NodeProxyTypeDaemonSet:
		objDs := obj.(*appsv1.DaemonSet)
		newDs := new.(*appsv1.DaemonSet)
		if reflect.DeepEqual(objDs.Spec.Selector, newDs.Spec.Selector) &&
			reflect.DeepEqual(objDs.Spec.Template.ObjectMeta, newDs.Spec.Template.ObjectMeta) &&
			reflect.DeepEqual(objDs.Spec.Template.Spec.Containers, newDs.Spec.Template.Spec.Containers) &&
			reflect.DeepEqual(objDs.Spec.Template.Spec.InitContainers, newDs.Spec.Template.Spec.InitContainers) &&
			reflect.DeepEqual(objDs.OwnerReferences, newDs.OwnerReferences) {
			for key, value := range newDs.Annotations {
				if objvalue, exist := objDs.Annotations[key]; !exist || objvalue != value {
					return true
				}
			}
			return false
		}
	}

	klog.V(4).Info("node-proxy deploy/ds changed, need to update it.")
	return true
}

// Called when OpenELB NodeProxy Service was deleted
func (r *ServiceReconciler) reconcileNPDelete(ctx context.Context, svc *corev1.Service) (ctrl.Result, error) {
	klog.V(4).Infof("Reconciling deletion %s/%s finalizing", svc.GetNamespace(), svc.GetName())
	var err error

	if util.ContainsString(svc.GetFinalizers(), constant.NodeProxyFinalizerName) {
		dpDsNamespacedName := types.NamespacedName{Namespace: util.EnvNamespace(), Name: proxyRescName(svc.Name, svc.Namespace)}
		var proxyResc client.Object
		nptypes := svc.Annotations[constant.NodeProxyTypeAnnotationKey]
		switch nptypes {
		case constant.NodeProxyTypeDeployment:
			proxyResc = &appsv1.Deployment{}
		case constant.NodeProxyTypeDaemonSet:
			proxyResc = &appsv1.DaemonSet{}
		default:
			klog.Info("unsupport OpenELB NodeProxy annotation value:" + nptypes)
			return ctrl.Result{}, nil
		}
		err = r.Get(ctx, dpDsNamespacedName, proxyResc)
		if err != nil {
			if errors.IsNotFound(err) {
				return ctrl.Result{}, nil
			}
			klog.Errorf("can't get deployment/daemonset for deletion: %v", err)
			return ctrl.Result{}, err
		}
		if err = r.Delete(ctx, proxyResc); err != nil {
			klog.Errorf("can't remove deployment/daemonset: %v", err)
			return ctrl.Result{}, err
		}
		klog.Infof("deleting node-proxy %s/%s", nptypes, proxyResc.GetName())
		controllerutil.RemoveFinalizer(svc, constant.NodeProxyFinalizerName)
		if err = r.Update(ctx, svc); err != nil {
			klog.Errorf("can't remove finalizer: %v", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) reconcileNP(ctx context.Context, svc *corev1.Service) (ctrl.Result, error) {
	if svc.DeletionTimestamp.IsZero() {
		return r.reconcileNPNormal(ctx, svc)
	}
	return r.reconcileNPDelete(ctx, svc)
}

// Judge whether this load balancer should be exposed by OpenELB Service
// Such Service will be exposed by Proxy Pod
func IsOpenELBNPService(obj runtime.Object) bool {
	if svc, ok := obj.(*corev1.Service); ok {
		return validate.HasOpenELBAnnotation(svc.Annotations) && validate.IsTypeLoadBalancer(svc) && validate.HasOpenELBNPAnnotation(svc.Annotations)
	}
	return false
}

func (r *ServiceReconciler) cleanNodeProxyData(ctx context.Context, svc *corev1.Service) (ctrl.Result, error) {
	_, exist := svc.Annotations[constant.NodeProxyTypeAnnotationKey]
	if exist {
		return r.reconcileNPDelete(ctx, svc)
	}

	dpDsNamespacedName := types.NamespacedName{Namespace: util.EnvNamespace(), Name: proxyRescName(svc.Name, svc.Namespace)}
	proxyDeploy := &appsv1.Deployment{}
	if err := r.Get(ctx, dpDsNamespacedName, proxyDeploy); err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("can't get deployment/daemonset for deletion: %v", err)
			return ctrl.Result{}, err
		}
	} else {
		if err = r.Delete(ctx, proxyDeploy); err != nil {
			klog.Errorf("can't remove deployment/daemonset: %v", err)
			return ctrl.Result{}, err
		}
		klog.Infof("deleting node-proxy deployment/%s", proxyDeploy.GetName())
	}

	proxyDs := &appsv1.DaemonSet{}
	if err := r.Get(ctx, dpDsNamespacedName, proxyDs); err != nil {
		if !errors.IsNotFound(err) {
			klog.Errorf("can't get deployment/daemonset for deletion: %v", err)
			return ctrl.Result{}, err
		}
	} else {
		if err = r.Delete(ctx, proxyDs); err != nil {
			klog.Errorf("can't remove deployment/daemonset: %v", err)
			return ctrl.Result{}, err
		}
		klog.Infof("deleting node-proxy daemonset/%s", proxyDs.GetName())
	}

	if util.ContainsString(svc.GetFinalizers(), constant.NodeProxyFinalizerName) {
		controllerutil.RemoveFinalizer(svc, constant.NodeProxyFinalizerName)
	}

	return ctrl.Result{}, nil
}
