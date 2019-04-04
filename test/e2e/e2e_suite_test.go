package e2e_test

import (
	"path"
	"runtime"
	"testing"

	"github.com/kubesphere/porter/pkg/apis"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	testClient    client.Client
	cfg           *rest.Config
	workspace     string
	testNamespace string
)

const (
	managerPodName = "controller-manager-0"
	managerName    = "controller-manager"
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func() {
	// testNamespace = os.Getenv("TEST_NS")
	// Expect(testNamespace).ShouldNot(BeEmpty())
	testNamespace = "porter-test-8cc68a84"
	cfg, err := config.GetConfig()
	Expect(err).ShouldNot(HaveOccurred(), "Error reading kubeconfig")
	apis.AddToScheme(scheme.Scheme)
	c, err := client.New(cfg, client.Options{})
	Expect(err).NotTo(HaveOccurred(), "Error in creating client")
	testClient = c
})

func getWorkspace() string {
	_, filename, _, _ := runtime.Caller(0)
	return path.Dir(filename)
}
