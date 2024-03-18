package framework

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/onsi/ginkgo/v2"
	"github.com/openelb/openelb/api/v1alpha2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/test/e2e/framework"
	admissionapi "k8s.io/pod-security-admission/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KubesphereDescribe annotates the test with the SIG label.
func KubesphereDescribe(text string, body func()) bool {
	return ginkgo.Describe("[LB:OpenELB] "+text, body)
}

type Framework struct {
	KubeConfig string
	*framework.Framework
	Scheme        *runtime.Scheme
	OpenELBClient client.Client
}

func NewDefaultFramework(baseName string) *Framework {
	scheme := runtime.NewScheme()

	if err := corev1.AddToScheme(scheme); err != nil {
		Failf("unable add kubernetes core APIs to scheme: %v", err)
	}

	if err := appsv1.AddToScheme(scheme); err != nil {
		Failf("unable add kubernetes apps APIs to scheme: %v", err)
	}

	if err := v1alpha2.AddToScheme(scheme); err != nil {
		Failf("unable add openelb APIs to scheme: %v", err)
	}

	f := &Framework{
		Scheme:     scheme,
		KubeConfig: framework.TestContext.KubeConfig,
		Framework:  framework.NewDefaultFramework(baseName),
	}

	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged
	ginkgo.BeforeEach(f.BeforeEach)

	return f
}

// BeforeEach gets a openelb client
func (f *Framework) BeforeEach() {
	//config openelb client
	if f.OpenELBClient == nil {
		config, err := framework.LoadConfig()
		ExpectNoError(err)

		config.QPS = f.Options.ClientQPS
		config.Burst = f.Options.ClientBurst
		f.OpenELBClient, err = client.New(config, client.Options{
			Scheme: f.Scheme,
		})
		framework.ExpectNoError(err)
	}
}
