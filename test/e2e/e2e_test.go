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

var templateTestCase e2eutil.TestCase

func init() {
	workspace = getWorkspace() + "/../.."
}

var _ = Describe("e2e", func() {
	serviceTypes := types.NamespacedName{Namespace: "default", Name: "mylbapp"}
	BeforeEach(func() {
		templateTestCase = e2eutil.TestCase{
			ControllerAS:           65000,
			RouterAS:               65001,
			ControllerPort:         179,
			ControllerIP:           "192.168.98.2",
			RouterTemplatePath:     workspace + "/test/test-configs/reciever.template",
			ControllerTemplatePath: workspace + "/test/test-configs/sender.template",
			KustomizePath:          workspace + "/config/default/",
			Namespace:              testNamespace,
			K8sClient:              testClient,
			RouterConfigPath:       "/root/bgp/test.toml",
			DeployYamlPath:         workspace + "/deploy/porter.yaml",
		}
	})
	It("Should write iptables when using portforword mode", func() {
		thisTestCase := templateTestCase
		thisTestCase.UsePortForward = true
		thisTestCase.ControllerPort = 17900
		thisTestCase.RouterIP = "192.168.98.8"

		containerIdCh := make(chan string)
		defer close(containerIdCh)
		errCh := make(chan error)
		defer close(errCh)

		go thisTestCase.StartRemoteRoute(containerIdCh, errCh)
		Eventually(errCh, 10*time.Second).ShouldNot(Receive())
		var containerID string
		Eventually(containerIdCh, 10*time.Second).Should(Receive(&containerID))
		thisTestCase.SetRouterContainerID(containerID)

		defer Expect(thisTestCase.StopRouter()).ShouldNot(HaveOccurred(), "Failed to stop bgp")
		//apply yaml
		Expect(thisTestCase.DeployYaml(workspace+"/config/bgp/config.toml")).ShouldNot(HaveOccurred(), "Failed to deploy yaml")
		defer Expect(e2eutil.KubectlDelete(thisTestCase.DeployYamlPath)).ShouldNot(HaveOccurred(), "Failed to delete yaml")

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
	FIt("Should work well when using samples", func() {
		thisTestCase := templateTestCase
		thisTestCase.RouterIP = "192.168.98.8"
		Expect(thisTestCase.StartRemoteRoute()).NotTo(HaveOccurred(), "Error in starting remote bgp")
		defer Expect(thisTestCase.StopRouter()).ShouldNot(HaveOccurred(), "Faild to stop container")
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
			Expect(testClient.Delete(context.TODO(), eip)).ShouldNot(HaveOccurred())
			Expect(e2eutil.WaitForDeletion(testClient, eip, 5*time.Second, 1*time.Minute)).ShouldNot(HaveOccurred())
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
		}, time.Minute, 2*time.Second).Should(Succeed())
		log, err := e2eutil.CheckManagerLog(testNamespace, managerName)
		Expect(err).ShouldNot(HaveOccurred(), log)
		log, err = e2eutil.CheckAgentLog(testNamespace, "porter-agent", testClient)
		Expect(err).ShouldNot(HaveOccurred(), log)
	})
})

// var _ = Describe("E2e", func() {
// 	serviceTypes := types.NamespacedName{Namespace: "default", Name: "mylbapp"}
// 	var noBGPPort string = "17900"
// 	var birdIP string = os.Getenv("BIRD_IP")
// 	It("Should write iptables when using portforword mode", func() {
// 		//get master node
// 		//read config if we need test
// 		if !testBGPConfig.PorterConfig.UsingPortForward {
// 			return
// 		}
// 		if birdIP != "" {
// 			pod := &corev1.Pod{}
// 			Expect(testClient.Get(context.TODO(), types.NamespacedName{Namespace: testNamespace, Name: managerPodName}, pod)).ShouldNot(HaveOccurred())
// 			nodeIP := pod.Status.HostIP
// 			output, err := e2eutil.QuickConnectAndRun(nodeIP, "iptables -nL PREROUTING -t nat | grep "+noBGPPort)
// 			Expect(err).NotTo(HaveOccurred(), "Error in listing NAT tables")
// 			Expect(output).To(ContainSubstring(birdIP))
// 			Expect(output).To(ContainSubstring(fmt.Sprintf("to:%s:%s", nodeIP, noBGPPort)))
// 			//check SNAT
// 			output, err = e2eutil.QuickConnectAndRun(nodeIP, "iptables -nL POSTROUTING -t nat | grep "+noBGPPort)
// 			Expect(err).NotTo(HaveOccurred(), "Error in listing NAT tables")
// 			Expect(string(output)).To(ContainSubstring("MASQUERADE"))
// 			Expect(string(output)).To(ContainSubstring(nodeIP))
// 		}
// 	})
// 	It("Should get right endpoints", func() {
// 		cmd := exec.Command("kubectl", "apply", "-f", workspace+"/config/samples/service.yaml")
// 		Expect(cmd.Run()).ShouldNot(HaveOccurred())
// 		service := &corev1.Service{}
// 		Eventually(func() error {
// 			err := testClient.Get(context.TODO(), serviceTypes, service)
// 			return err
// 		}, time.Second*20, time.Second).Should(Succeed())
// 		defer deleteServiceGracefully(service)

// 		Eventually(func() int {
// 			ips, err := kubeutil.GetServiceNodesIP(testClient, service)
// 			if err != nil {
// 				fmt.Println("Falied to get ips using client")
// 				return 0
// 			}
// 			//fmt.Fprintln(GinkgoWriter, ips)
// 			return len(ips)
// 		}, time.Minute, time.Second*2).Should(BeNumerically(">=", 2))
// 	})

// 	It("Should work well when using samples", func() {
// 		eip := &networkv1alpha1.EIP{}
// 		reader, err := os.Open(workspace + "/config/samples/network_v1alpha1_eip.yaml")
// 		Expect(err).NotTo(HaveOccurred(), "Cannot read sample yamls")
// 		err = yaml.NewYAMLOrJSONDecoder(reader, 10).Decode(eip)
// 		Expect(err).NotTo(HaveOccurred(), "Cannot unmarshal yamls")
// 		if eip.Namespace == "" {
// 			eip.Namespace = "default"
// 		}
// 		err = testClient.Create(context.TODO(), eip)
// 		Expect(err).NotTo(HaveOccurred())
// 		defer func() {
// 			Expect(testClient.Delete(context.TODO(), eip)).ShouldNot(HaveOccurred())
// 			Expect(e2eutil.WaitForDeletion(testClient, eip, 5*time.Second, 1*time.Minute)).ShouldNot(HaveOccurred())
// 		}()

// 		//apply service
// 		cmd := exec.Command("kubectl", "apply", "-f", workspace+"/config/samples/service.yaml")
// 		Expect(cmd.Run()).ShouldNot(HaveOccurred())
// 		service := &corev1.Service{}
// 		Eventually(func() error {
// 			err := testClient.Get(context.TODO(), serviceTypes, service)
// 			return err
// 		}, time.Second*30, 5*time.Second).Should(Succeed())
// 		defer deleteServiceGracefully(service)

// 		//Service should get its eip
// 		Eventually(func() error {
// 			service := &corev1.Service{}
// 			err := testClient.Get(context.TODO(), serviceTypes, service)
// 			if err != nil {
// 				return err
// 			}
// 			if len(service.Status.LoadBalancer.Ingress) > 0 && service.Status.LoadBalancer.Ingress[0].IP == eip.Spec.Address {
// 				return nil
// 			}
// 			return fmt.Errorf("Failed")
// 		}, 2*time.Minute, time.Second).Should(Succeed())
// 		//check route in bird
// 		if birdIP != "" {
// 			session, err := e2eutil.Connect("root", "", birdIP, e2eutil.GetDefaultPrivateKeyFile(), 22, nil)
// 			Expect(err).NotTo(HaveOccurred(), "Connect Bird using private key FAILED")
// 			defer session.Close()
// 			stdinBuf, err := session.StdinPipe()
// 			var outbt, errbt bytes.Buffer
// 			session.Stdout = &outbt
// 			session.Stderr = &errbt
// 			err = session.Shell()
// 			Expect(err).ShouldNot(HaveOccurred(), "Failed to start ssh shell")
// 			Eventually(func() error {
// 				stdinBuf.Write([]byte("ip route\n"))
// 				ips, err := e2eutil.GetServiceNodesIP(testClient, serviceTypes.Namespace, serviceTypes.Name)
// 				if err != nil {
// 					return err
// 				}
// 				s := outbt.String() + errbt.String()
// 				for _, ip := range ips {
// 					if !strings.Contains(s, fmt.Sprintf("nexthop via %s", ip)) {
// 						return fmt.Errorf("No routes in Brid")
// 					}
// 				}
// 				if strings.Contains(s, fmt.Sprintf("%s  proto bird", eip.Spec.Address)) {
// 					return nil
// 				} else {
// 					return fmt.Errorf("No routes in Brid")
// 				}
// 			}, time.Minute, 2*time.Second).Should(Succeed())
// 		}
// 	})
// })

func deleteServiceGracefully(service *corev1.Service) {
	Expect(e2eutil.KubectlDelete(workspace + "/config/samples/service.yaml")).ShouldNot(HaveOccurred())
	Expect(e2eutil.WaitForDeletion(testClient, service, time.Second*5, time.Minute)).ShouldNot(HaveOccurred(), "Failed waiting for services deletion")
}
