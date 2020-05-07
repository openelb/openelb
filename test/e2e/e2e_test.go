package e2e_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	bgpserver "github.com/kubesphere/porter/pkg/bgp/serverd"
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
	t.ControllerAS = 65000
	t.RouterAS = 65001
	t.ControllerPort = 17900
	t.ControllerIP = os.Getenv("MASTER_IP")
	t.RouterIP = os.Getenv("ROUTER_IP")
	t.K8sClient = testClient
	t.Namespace = testNamespace
	t.RouterConfigPath = "/root/bgp/router.toml"
	t.RouterTemplatePath = workspace + "/test/test-configs/reciever.template"
	return t
}

var _ = Describe("e2e", func() {
	//serviceTypes := types.NamespacedName{Namespace: "default", Name: "mylbapp"}

	It("Should write iptables when using portforword mode", func() {
		thisTestCase := GetDefaultTestCase("portforward")

		bgpPeer := &networkv1alpha1.BgpPeer{
			Spec: bgpserver.BgpPeerSpec{
				Config: bgpserver.NeighborConfig{
					PeerAs:          uint32(thisTestCase.RouterAS),
					NeighborAddress: thisTestCase.RouterIP,
				},
				AddPaths:         bgpserver.AddPaths{},
				Transport:        bgpserver.Transport{},
				UsingPortForward: true,
			},
		}
		bgpPeer.Name = "test-peer"
		Expect(thisTestCase.K8sClient.Create(context.TODO(), bgpPeer)).NotTo(HaveOccurred())
		defer func() {
			thisTestCase.K8sClient.Delete(context.TODO(), bgpPeer)
			e2eutil.WaitForDeletion(thisTestCase.K8sClient, bgpPeer, 5*time.Second, 1*time.Minute)
		}()

		bgpConf := &networkv1alpha1.BgpConf{
			Spec: bgpserver.BgpConfSpec{
				Port:     int32(thisTestCase.ControllerPort),
				As:       uint32(thisTestCase.ControllerAS),
				RouterId: thisTestCase.ControllerIP,
			},
		}
		bgpConf.Name = "test-bgpconf"
		Expect(thisTestCase.K8sClient.Create(context.TODO(), bgpConf)).NotTo(HaveOccurred())
		defer func() {
			thisTestCase.K8sClient.Delete(context.TODO(), bgpConf)
			e2eutil.WaitForDeletion(thisTestCase.K8sClient, bgpConf, 5*time.Second, 1*time.Minute)
		}()

		Expect(thisTestCase.StartRemoteRoute()).NotTo(HaveOccurred(), "Error in starting remote bgp")
		defer thisTestCase.StopRouter()

		podlist := &corev1.PodList{}
		Expect(testClient.List(context.TODO(), podlist, client.InNamespace(thisTestCase.Namespace), client.MatchingLabels{"app": "porter-manager"})).ShouldNot(HaveOccurred())
		nodeIP := podlist.Items[0].Status.HostIP
		output, err := e2eutil.QuickConnectAndRun(nodeIP, "iptables -nL PREROUTING -t nat | grep "+strconv.Itoa(thisTestCase.ControllerPort))
		Expect(err).NotTo(HaveOccurred(), "Error in listing NAT tables")
		Expect(output).To(ContainSubstring(thisTestCase.RouterIP))
		Expect(output).To(ContainSubstring(fmt.Sprintf("to:%s:%d", "", thisTestCase.ControllerPort)))

		//CheckLog
		log, err := thisTestCase.GetRouterLog()
		Expect(err).ShouldNot(HaveOccurred(), "Failed to get log of router")
		Expect(log).ShouldNot(ContainSubstring("error"))
	})

	It("Should work well in passive mode when using samples", func() {
		thisTestCase := GetDefaultTestCase("passivemode")
		thisTestCase.IsPassiveMode = true

		thisTestCase.StartDefaultTest(workspace)
	})

	It("Should work well in layer2 mode when using samples", func() {
		thisTestCase := GetDefaultTestCase("layer2")
		thisTestCase.Layer2 = true

		thisTestCase.StartDefaultTest(workspace)
	})

	It("Should work well when using samples", func() {
		thisTestCase := GetDefaultTestCase("sample")

		thisTestCase.InjectTest = func() {
			checkFn := func() {
				deploy := &appsv1.Deployment{}
				err := thisTestCase.K8sClient.Get(context.TODO(), types.NamespacedName{Name: "test-app", Namespace: testNamespace}, deploy)
				Expect(err).ShouldNot(HaveOccurred())
				rep := int32(1)
				deploy.Spec.Replicas = &rep
				err = thisTestCase.K8sClient.Update(context.TODO(), deploy)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(func() int {
					s, err := thisTestCase.CheckBGPRoute()
					log.Printf("route %v", s)
					if err == nil {
						s = strings.TrimSpace(s)
						return len(strings.Split(s, "\n")) - 1
					}
					log.Println("Failed to get route in bgp, err: " + err.Error())
					return 0
				}, time.Second*30, 5*time.Second).Should(BeEquivalentTo(rep))
			}
			checkFn()
		}
		thisTestCase.StartDefaultTest(workspace)
	})
})
