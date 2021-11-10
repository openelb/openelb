package constant

const (
	FinalizerName     string = "finalizer.lb.kubesphere.io/v1alpha1"
	IPAMFinalizerName string = "finalizer.ipam.kubesphere.io/v1alpha1"

	// When used for annotation, it means that the service address is assigned by the porter
	// When used as a label, it indicates on which node the porter manager is deployed
	PorterAnnotationKey   string = "lb.kubesphere.io/v1alpha1"
	PorterAnnotationValue string = "porter"

	// Indicates the node to which layer2 traffic is sent
	PorterLayer2Annotation string = "layer2.porter.kubesphere.io/v1alpha1"

	NodeProxyTypeAnnotationKey        string = "node-proxy.porter.kubesphere.io/type"
	NodeProxyTypeDeployment           string = "deployment"
	NodeProxyTypeDaemonSet            string = "daemonset"
	LabelNodeProxyExternalIPPreffered string = "node-proxy.porter.kubesphere.io/external-ip-preffered"
	LabelNodeProxyExcludeNode         string = "node-proxy.porter.kubesphere.io/exclude-node"
	NodeProxyExternalIPAnnotationKey  string = "node-proxy.porter.kubesphere.io/external-ip"
	NodeProxyInternalIPAnnotationKey  string = "node-proxy.porter.kubesphere.io/internal-ip"
	NodeProxyDefaultForwardImage      string = "kubespheredev/openelb-forward:v0.4.2"
	NodeProxyDefaultProxyImage        string = "kubespheredev/openelb-proxy:v0.4.2"
	NameSeparator                     string = "-"
	IPSeparator                       string = ","
	EnvArgSplitter                    string = " "
	NodeProxyWorkloadPrefix           string = "node-proxy-"
	NodeProxyFinalizerName            string = "node-proxy.porter.kubesphere.io/finalizer"
	NodeProxyConfigMapName            string = "node-proxy-config"
	NodeProxyConfigMapForwardImage    string = "forward-image"
	NodeProxyConfigMapProxyImage      string = "proxy-image"

	KubernetesMasterLabel string = "node-role.kubernetes.io/master"

	PorterEIPAnnotationKey         string = "eip.porter.kubesphere.io/v1alpha1"
	PorterEIPAnnotationKeyV1Alpha2 string = "eip.porter.kubesphere.io/v1alpha2"

	PorterProtocolAnnotationKey string = "protocol.porter.kubesphere.io/v1alpha1"

	PorterNodeRack string = "porter.kubesphere.io/rack"
	// TODO: Disable lable modification using webhook
	PorterCNI string = "porter.kubesphere.io/cni"

	PorterProtocolBGP    string = "bgp"
	PorterProtocolLayer2 string = "layer2"
	PorterProtocolDummy  string = "dummy"
	PorterCNICalico      string = "calico"
	EipRangeSeparator    string = "-"

	PorterSpeakerLocker = "porter-speaker"
	PorterNamespace     = "porter-system"

	EnvPorterNamespace = "PORTER_NAMESPACE"
	EnvNodeName        = "NODE_NAME"
)
