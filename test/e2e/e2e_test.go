package e2e_test

import (
	"context"
	"fmt"
	"strconv"

	"github.com/kubesphere/porter/test/e2eutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func init() {
	workspace = getWorkspace() + "/../.."
}

func GetDefaultTestCase(name string) *e2eutil.TestCase {
	t := new(e2eutil.TestCase)
	t.Name = name
	t.ControllerAS = 65000
	t.RouterAS = 65001
	t.ControllerPort = 179
	t.DeployYamlPath = workspace + "/deploy/porter.yaml"
	t.K8sClient = testClient
	t.KustomizePath = workspace + "/config/default/"
	t.Namespace = testNamespace
	t.RouterConfigPath = "/root/bgp/test.toml"
	t.RouterTemplatePath = workspace + "/test/test-configs/reciever.template"
	t.ControllerTemplatePath = workspace + "/test/test-configs/sender.template"
	t.ControllerIP = "192.168.98.2"
	t.KustomizeConfigPath = workspace + "/config/bgp/config.toml"
	return t
}

var _ = Describe("e2e", func() {
	//serviceTypes := types.NamespacedName{Namespace: "default", Name: "mylbapp"}
	It("Should write iptables when using portforword mode", func() {
		thisTestCase := GetDefaultTestCase("portforward")
		thisTestCase.UsePortForward = true
		thisTestCase.ControllerPort = 17900
		thisTestCase.RouterIP = "192.168.98.8"

		Expect(thisTestCase.StartRemoteRoute()).NotTo(HaveOccurred(), "Error in starting remote bgp")
		defer thisTestCase.StopRouter()
		//apply yaml
		Expect(thisTestCase.DeployYaml()).ShouldNot(HaveOccurred(), "Failed to deploy yaml")
		defer e2eutil.KubectlDelete(thisTestCase.DeployYamlPath)

		pod := &corev1.Pod{}
		Expect(testClient.Get(context.TODO(), types.NamespacedName{Namespace: testNamespace, Name: managerPodName}, pod)).ShouldNot(HaveOccurred())
		nodeIP := pod.Status.HostIP
		output, err := e2eutil.QuickConnectAndRun(nodeIP, "iptables -nL PREROUTING -t nat | grep "+strconv.Itoa(thisTestCase.ControllerPort))
		Expect(err).NotTo(HaveOccurred(), "Error in listing NAT tables")
		Expect(output).To(ContainSubstring(thisTestCase.RouterIP))
		Expect(output).To(ContainSubstring(fmt.Sprintf("to:%s:%d", nodeIP, thisTestCase.ControllerPort)))
		//check SNAT
		output, err = e2eutil.QuickConnectAndRun(nodeIP, "iptables -nL POSTROUTING -t nat | grep "+strconv.Itoa(thisTestCase.ControllerPort))
		Expect(err).NotTo(HaveOccurred(), "Error in listing NAT tables")
		Expect(string(output)).To(ContainSubstring("MASQUERADE"))
		Expect(string(output)).To(ContainSubstring(nodeIP))

		//CheckLog
		log, err := thisTestCase.GetRouterLog()
		Expect(err).ShouldNot(HaveOccurred(), "Failed to get log of router")
		Expect(log).ShouldNot(ContainSubstring("error"))
	})
	It("Should work well when using samples", func() {
		thisTestCase := GetDefaultTestCase("sample")
		thisTestCase.RouterIP = "192.168.98.8"
		thisTestCase.StartDefaultTest(workspace)
	})
	It("Should work well in passive mode when using samples", func() {
		thisTestCase := GetDefaultTestCase("passivemode")
		thisTestCase.RouterIP = "192.168.98.8"
		thisTestCase.IsPassiveMode = true
		thisTestCase.StartDefaultTest(workspace)
	})
})
