package constant

const (
	FinalizerName             string = "finalizer.lb.kubesphere.io/v1alpha1"
	NodeFinalizerName         string = "finalizer.lb.kubesphere.io"
	IPAMFinalizerName         string = "finalizer.ipam.kubesphere.io/v1alpha1"
	PorterAnnotationKey       string = "lb.kubesphere.io/v1alpha1"
	PorterAnnotationValue     string = "porter"
	PorterEIPAnnotationKey    string = "eip.porter.kubesphere.io/v1alpha1"
	PorterLBTypeAnnotationKey string = "lbtype.porter.kubesphere.io/v1alpha1"
	PorterLBTypeBGP           string = "bgp"
	PorterLBTypeLayer2        string = "layer2"
)
