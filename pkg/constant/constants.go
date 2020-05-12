package constant

const (
	FinalizerName               string = "finalizer.lb.kubesphere.io/v1alpha1"
	NodeFinalizerName           string = "finalizer.lb.kubesphere.io"
	IPAMFinalizerName           string = "finalizer.ipam.kubesphere.io/v1alpha1"
	PorterAnnotationKey         string = "lb.kubesphere.io/v1alpha1"
	PorterAnnotationValue       string = "porter"
	PorterEIPAnnotationKey      string = "eip.porter.kubesphere.io/v1alpha1"
	PorterProtocolAnnotationKey string = "protocol.porter.kubesphere.io/v1alpha1"
	PorterProtocolBGP           string = "bgp"
	PorterProtocolLayer2        string = "layer2"
)
