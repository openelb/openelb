package e2e_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kubesphere/porter/test/e2eutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	t.ControllerIP = os.Getenv("MASTER_IP")
	t.DeployYamlPath = os.Getenv("YAML_PATH")
	t.K8sClient = testClient
	t.KustomizePath = workspace + "/config/dev/"
	t.Namespace = testNamespace
	t.RouterConfigPath = "/root/bgp/test.toml"
	t.RouterTemplatePath = workspace + "/test/test-configs/reciever.template"
	t.ControllerTemplatePath = workspace + "/test/test-configs/sender.template"
	t.ControllerConfigPath = "/tmp/config.toml"
	t.ControllerName = managerName
	t.TestDeploymentName = testDeployName
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
		defer func() {
			Expect(thisTestCase.DeleteController()).ShouldNot(HaveOccurred(), "Failed to delete controller")
		}()

		podlist := &corev1.PodList{}
		Expect(testClient.List(context.TODO(), podlist, client.InNamespace(thisTestCase.Namespace), client.MatchingLabels{"app": "porter-manager"})).ShouldNot(HaveOccurred())
		nodeIP := podlist.Items[0].Status.HostIP
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
		thisTestCase.RouterIP = os.Getenv("ROUTER_IP")
		thisTestCase.InjectTest = func() {
			incre := -1
			checkFn := func() {
				deploy := &appsv1.Deployment{}
				err := thisTestCase.K8sClient.Get(context.TODO(), types.NamespacedName{Name: testDeployName, Namespace: testNamespace}, deploy)
				Expect(err).ShouldNot(HaveOccurred())
				rep := *(deploy.Spec.Replicas)
				rep += int32(incre)
				deploy.Spec.Replicas = &rep
				err = thisTestCase.K8sClient.Update(context.TODO(), deploy)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(func() int {
					s, err := thisTestCase.CheckBGPRoute()
					if err == nil {
						s = strings.TrimSpace(s)
						return len(strings.Split(s, "\n")) - 1
					}
					log.Println("Failed to get route in bgp, err: " + err.Error())
					return 0
				}, time.Second*30, time.Second*5).Should(BeEquivalentTo(rep))
			}
			checkFn()
			incre = 1
			checkFn()
		}
		thisTestCase.StartDefaultTest(workspace)
	})
	It("Should work well in passive mode when using samples", func() {
		thisTestCase := GetDefaultTestCase("passivemode")
		thisTestCase.RouterIP = "192.168.98.8"
		thisTestCase.IsPassiveMode = true
		thisTestCase.StartDefaultTest(workspace)
	})
})
