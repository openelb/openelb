package manager

import (
	networkv1alpha2 "github.com/kubesphere/porter/api/v1alpha2"
	"github.com/kubesphere/porter/pkg/manager/client"
	"github.com/spf13/pflag"
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

type GenericOptions struct {
	MetricsAddr   string
	ReadinessAddr string
}

func NewGenericOptions() *GenericOptions {
	return &GenericOptions{
		MetricsAddr:   ":8080",
		ReadinessAddr: ":8000",
	}
}

func (options *GenericOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&options.MetricsAddr, "metrics-addr", options.MetricsAddr, "The address the metric endpoint binds to.")
	fs.StringVar(&options.ReadinessAddr, "readiness-addr", options.ReadinessAddr, "The address readinessProbe used")
}

func NewManager(cfg *rest.Config, options *GenericOptions) (ctrl.Manager, error) {
	opts := ctrl.Options{
		Scheme: scheme,
	}
	if options != nil {
		opts.MetricsBindAddress = options.MetricsAddr
	}
	result, err := ctrl.NewManager(cfg, opts)

	client.Client = result.GetClient()

	return result, err
}

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = corev1.AddToScheme(scheme)
	_ = admissionv1.AddToScheme(scheme)
	_ = admissionv1beta1.AddToScheme(scheme)
	_ = networkv1alpha2.AddToScheme(scheme)
}
