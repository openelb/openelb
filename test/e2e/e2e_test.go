package e2e_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	networkv1alpha1 "github.com/kubesphere/porter/pkg/apis/network/v1alpha1"
	"github.com/kubesphere/porter/test/e2eutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func init() {
	workspace = getWorkspace() + "/../.."
}

func GetDefaultTestCase() *e2eutil.TestCase {
	t := new(e2eutil.TestCase)
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
	return t
}

var _ = Describe("e2e", func() {
	serviceTypes := types.NamespacedName{Namespace: "default", Name: "mylbapp"}
	It("Should write iptables when using portforword mode", func() {
		thisTestCase := GetDefaultTestCase()
		thisTestCase.UsePortForward = true
		thisTestCase.ControllerPort = 17900
		thisTestCase.RouterIP = "192.168.98.8"

		Expect(thisTestCase.StartRemoteRoute()).NotTo(HaveOccurred(), "Error in starting remote bgp")
		defer thisTestCase.StopRouter()
		//apply yaml
		Expect(thisTestCase.DeployYaml(workspace+"/config/bgp/config.toml")).ShouldNot(HaveOccurred(), "Failed to deploy yaml")
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
		thisTestCase := GetDefaultTestCase()
		thisTestCase.RouterIP = "192.168.98.8"
		Expect(thisTestCase.StartRemoteRoute()).NotTo(HaveOccurred(), "Error in starting remote bgp")
		defer thisTestCase.StopRouter()
		//apply yaml
		Expect(thisTestCase.DeployYaml(workspace+"/config/bgp/config.toml")).ShouldNot(HaveOccurred(), "Failed to deploy yaml")
		defer e2eutil.KubectlDelete(thisTestCase.DeployYamlPath)

		//testing
		eip := &networkv1alpha1.EIP{}
		reader, err := os.Open(workspace + "/config/samples/network_v1alpha1_eip.yaml")
		Expect(err).NotTo(HaveOccurred(), "Cannot read sample yamls")
		err = yaml.NewYAMLOrJSONDecoder(reader, 10).Decode(eip)
		Expect(err).NotTo(HaveOccurred(), "Cannot unmarshal yamls")
		if eip.Namespace == "" {
			eip.Namespace = "default"
		}
		err = testClient.Create(context.TODO(), eip)
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			testClient.Delete(context.TODO(), eip)
			e2eutil.WaitForDeletion(testClient, eip, 5*time.Second, 1*time.Minute)
		}()

		//apply service
		Expect(e2eutil.KubectlApply(workspace + "/config/samples/service.yaml")).ShouldNot(HaveOccurred())
		service := &corev1.Service{}
		Eventually(func() error {
			err := testClient.Get(context.TODO(), serviceTypes, service)
			return err
		}, time.Second*30, 5*time.Second).Should(Succeed())
		defer deleteServiceGracefully(service)

		//Service should get its eip
		Eventually(func() error {
			service := &corev1.Service{}
			err := testClient.Get(context.TODO(), serviceTypes, service)
			if err != nil {
				return err
			}
			if len(service.Status.LoadBalancer.Ingress) > 0 && service.Status.LoadBalancer.Ingress[0].IP == eip.Spec.Address {
				return nil
			}
			return fmt.Errorf("Failed")
		}, 2*time.Minute, time.Second).Should(Succeed())
		//check route in bird
		if thisTestCase.IsLocal() {
			Eventually(func() error {
				s, err := e2eutil.CheckBGPRoute()
				if err != nil {
					return err
				}
				ips, err := e2eutil.GetServiceNodesIP(testClient, serviceTypes.Namespace, serviceTypes.Name)
				if err != nil {
					return err
				}
				if len(ips) < 2 {
					return fmt.Errorf("Service Not Ready")
				}
				for _, ip := range ips {
					if !strings.Contains(s, ip) {
						return fmt.Errorf("No routes in GoBGP")
					}
				}
				return nil
			}, time.Minute, 5*time.Second).Should(Succeed())
		} else {
			session, err := e2eutil.QuickConnectUsingDefaultSSHKey(thisTestCase.RouterIP)
			Expect(err).NotTo(HaveOccurred(), "Connect Bird using private key FAILED")
			defer session.Close()
			stdinBuf, err := session.StdinPipe()
			var outbt, errbt bytes.Buffer
			session.Stdout = &outbt
			session.Stderr = &errbt
			err = session.Shell()
			Expect(err).ShouldNot(HaveOccurred(), "Failed to start ssh shell")
			Eventually(func() error {
				stdinBuf.Write([]byte("gobgp global rib\n"))
				ips, err := e2eutil.GetServiceNodesIP(testClient, serviceTypes.Namespace, serviceTypes.Name)
				if err != nil {
					return err
				}
				s := outbt.String() + errbt.String()
				for _, ip := range ips {
					if !strings.Contains(s, ip) {
						return fmt.Errorf("No routes in GoBGP")
					}
				}
				return nil
			}, time.Minute, 5*time.Second).Should(Succeed())
		}
		//CheckLog
		log, err := thisTestCase.GetRouterLog()
		Expect(err).ShouldNot(HaveOccurred(), "Failed to get log of router")
		Expect(log).ShouldNot(ContainSubstring("error"))

		log, err = e2eutil.CheckManagerLog(testNamespace, managerName)
		Expect(err).ShouldNot(HaveOccurred(), log)
		log, err = e2eutil.CheckAgentLog(testNamespace, "porter-agent", testClient)
		Expect(err).ShouldNot(HaveOccurred(), log)
	})
})

func deleteServiceGracefully(service *corev1.Service) {
	Expect(e2eutil.KubectlDelete(workspace + "/config/samples/service.yaml")).ShouldNot(HaveOccurred())
	Expect(e2eutil.WaitForDeletion(testClient, service, time.Second*5, time.Minute)).ShouldNot(HaveOccurred(), "Failed waiting for services deletion")
}
