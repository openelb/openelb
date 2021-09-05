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

	PorterLBSAnnotationKey     string = "lbs.porter.kubesphere.io/v1alpha1"
	PorterOnePort              string = "one"
	PorterAllNode              string = "all"
	PorterNodeExtnlIPPrefLabel string = "lbs.porter.kubesphere.io/external-ip-preffered"
	PorterNodeExcludeLBSLabel  string = "lbs.porter.kubesphere.io/exclude.node"
	PorterLBSExposedExternalIP string = "lbs.porter.kubesphere.io/exposed.node.external-ips"
	PorterLBSExposedInternalIP string = "lbs.porter.kubesphere.io/exposed.node.internal-ips"
	PorterForwardImage         string = "kony168/openelb-forward:v0.4.2"
	PorterProxyImage           string = "kony168/openelb-proxy:v0.4.2"
	NameSeparator              string = "-"
	IPSeparator                string = ","
	PorterDeDsPrefix           string = "svc-proxy-"
	PorterLBSFInalizer         string = "lbs.porter.kubesphere.io/finalizer"

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
