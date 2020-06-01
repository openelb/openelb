package e2e_test

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/kubesphere/porter/api/v1alpha1"
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
	managerName = "porter-manager"
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func() {
	testNamespace = os.Getenv("TEST_NS")
	Expect(testNamespace).ShouldNot(BeEmpty())
	cfg, err := config.GetConfig()
	Expect(err).ShouldNot(HaveOccurred(), "Error reading kubeconfig")
	Expect(v1alpha1.AddToScheme(scheme.Scheme)).ShouldNot(HaveOccurred())
	testClient, err = client.New(cfg, client.Options{})
	Expect(err).NotTo(HaveOccurred(), "Error in creating client")
})

var _ = AfterSuite(func() {
})

func getWorkspace() string {
	_, filename, _, _ := runtime.Caller(0)
	return path.Dir(filename)
}
