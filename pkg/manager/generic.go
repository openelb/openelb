package manager

import (
	networkv1alpha2 "github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/manager/client"
	"github.com/spf13/pflag"
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	nc "sigs.k8s.io/controller-runtime/pkg/client"
)

type GenericOptions struct {
	WebhookPort    int
	MetricsAddr    string
	ReadinessAddr  string
	KeepalivedPort int
}

func NewGenericOptions() *GenericOptions {
	return &GenericOptions{
		WebhookPort:   443,
		MetricsAddr:   ":50052",
		ReadinessAddr: "0",
		KeepalivedPort: 8080,
	}
}

func (options *GenericOptions) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&options.WebhookPort, "webhook-port", options.WebhookPort, "The port that the webhook server serves at")
	fs.StringVar(&options.MetricsAddr, "metrics-addr", options.MetricsAddr, "The address the metric endpoint binds to.")
	fs.StringVar(&options.ReadinessAddr, "readiness-addr", options.ReadinessAddr, "The address readinessProbe used")
	fs.IntVar(&options.KeepalivedPort, "keepalived-http-port", options.KeepalivedPort, "The port number used by keepalived for the http-port")
}

func NewManager(cfg *rest.Config, options *GenericOptions) (ctrl.Manager, error) {
	opts := ctrl.Options{
		Scheme: scheme,
	}
	if options != nil {
		opts.Port = options.WebhookPort
		opts.MetricsBindAddress = options.MetricsAddr
	}
	result, err := ctrl.NewManager(cfg, opts)

	if err == nil {
		client.Client, err = nc.New(cfg, nc.Options{Scheme: scheme})
	}

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
	_ = appsv1.AddToScheme(scheme)
}
