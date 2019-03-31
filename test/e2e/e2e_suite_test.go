package e2e_test

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/kubesphere/porter/pkg/apis"
	"github.com/kubesphere/porter/test/e2eutil"
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
	restClient    *rest.RESTClient
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
	//install deploy
	//install operator is writing in the `make debug`, maybe we should write here to decouple
	testNamespace = os.Getenv("TEST_NS")
	Expect(testNamespace).ShouldNot(BeEmpty())
	workspace = getWorkspace() + "/../.."
	cfg, err := config.GetConfig()
	Expect(err).ShouldNot(HaveOccurred(), "Error reading kubeconfig")
	apis.AddToScheme(scheme.Scheme)
	c, err := client.New(cfg, client.Options{})
	Expect(err).NotTo(HaveOccurred(), "Error in creating client")
	testClient = c

	fmt.Fprintln(GinkgoWriter, "cleaning up before running test")
	Expect(e2eutil.CleanEIPList(c)).ShouldNot(HaveOccurred(), "Cleanup failed")
	//waiting for controller up
	err = e2eutil.WaitForController(c, testNamespace, managerName, 5*time.Second, 2*time.Minute)
	Expect(err).ShouldNot(HaveOccurred(), "timeout waiting for controller up: %s\n", err)
	//waiting for bgp
	fmt.Fprintln(GinkgoWriter, "Controller is up now")

})

var _ = AfterSuite(func() {
	fmt.Fprintln(GinkgoWriter, "Begin to cleaning")
	//check logs
	log, err := e2eutil.CheckManagerLog(testNamespace, managerName)
	Expect(err).ShouldNot(HaveOccurred(), log)
	log, err = e2eutil.CheckAgentLog(testNamespace, "porter-agent", testClient)
	Expect(err).ShouldNot(HaveOccurred(), log)

	cmd := exec.Command("kubectl", "delete", "-f", workspace+"/deploy/porter.yaml")
	Expect(cmd.Run()).ShouldNot(HaveOccurred())
})

func getWorkspace() string {
	_, filename, _, _ := runtime.Caller(0)
	return path.Dir(filename)
}
