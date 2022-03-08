package constant

const (
	FinalizerName     string = "finalizer.lb.kubesphere.io/v1alpha1"
	IPAMFinalizerName string = "finalizer.ipam.kubesphere.io/v1alpha1"

	// When used for annotation, it means that the service address is assigned by the openelb
	// When used as a label, it indicates on which node the openelb manager is deployed
	OpenELBAnnotationKey   string = "lb.kubesphere.io/v1alpha1"
	OpenELBAnnotationValue string = "openelb"

	// Indicates the node to which layer2 traffic is sent
	OpenELBLayer2Annotation string = "layer2.openelb.kubesphere.io/v1alpha1"

	NodeProxyTypeAnnotationKey        string = "node-proxy.openelb.kubesphere.io/type"
	NodeProxyTypeDeployment           string = "deployment"
	NodeProxyTypeDaemonSet            string = "daemonset"
	LabelNodeProxyExternalIPPreffered string = "node-proxy.openelb.kubesphere.io/external-ip-preffered"
	LabelNodeProxyExcludeNode         string = "node-proxy.openelb.kubesphere.io/exclude-node"
	NodeProxyExternalIPAnnotationKey  string = "node-proxy.openelb.kubesphere.io/external-ip"
	NodeProxyInternalIPAnnotationKey  string = "node-proxy.openelb.kubesphere.io/internal-ip"
	NodeProxyDefaultForwardImage      string = "kubespheredev/openelb-forward:v0.4.2"
	NodeProxyDefaultProxyImage        string = "kubespheredev/openelb-proxy:v0.4.2"
	NameSeparator                     string = "-"
	IPSeparator                       string = ","
	EnvArgSplitter                    string = " "
	NodeProxyWorkloadPrefix           string = "node-proxy-"
	NodeProxyFinalizerName            string = "node-proxy.openelb.kubesphere.io/finalizer"
	NodeProxyConfigMapName            string = "node-proxy-config"
	NodeProxyConfigMapForwardImage    string = "forward-image"
	NodeProxyConfigMapProxyImage      string = "proxy-image"

	KubernetesMasterLabel string = "node-role.kubernetes.io/master"

	OpenELBEIPAnnotationKey         string = "eip.openelb.kubesphere.io/v1alpha1"
	OpenELBEIPAnnotationKeyV1Alpha2 string = "eip.openelb.kubesphere.io/v1alpha2"
	OpenELBEIPAnnotationDefaultPool string = "eip.openelb.kubesphere.io/is-default-eip"
	OpenELBProtocolAnnotationKey    string = "protocol.openelb.kubesphere.io/v1alpha1"

	OpenELBNodeRack string = "openelb.kubesphere.io/rack"
	// TODO: Disable lable modification using webhook
	OpenELBCNI string = "openelb.kubesphere.io/cni"

	OpenELBProtocolBGP    string = "bgp"
	OpenELBProtocolLayer2 string = "layer2"
	OpenELBProtocolDummy  string = "dummy"
	OpenELBProtocolVip    string = "vip"
	OpenELBCNICalico      string = "calico"
	EipRangeSeparator     string = "-"

	OpenELBSpeakerLocker      = "openelb-speaker"
	OpenELBNamespace          = "openelb-system"
	OpenELBConfigMap          = "openelb-vip-configmap"
	OpenELBVipName            = "openelb-keepalive-vip"
	OpenELBServiceAccountName = "kube-keepalived-vip"
	OpenELBKeepAliveImageName = "aledbf/kube-keepalived-vip:0.35"
	EnvOpenELBNamespace       = "OPENELB_NAMESPACE"
	EnvNodeName               = "NODE_NAME"
)
