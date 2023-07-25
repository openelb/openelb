package e2e

import (
	"context"
	"fmt"
	"net"
	"time"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"

	"github.com/onsi/ginkgo/v2"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/test/e2e/framework"
	v1 "k8s.io/api/core/v1"

	// rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/runtime/schema"
	// "k8s.io/apiserver/pkg/authentication/serviceaccount"
	clientset "k8s.io/client-go/kubernetes"
	// "k8s.io/kubernetes/test/e2e/framework/auth"

	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2eservice "k8s.io/kubernetes/test/e2e/framework/service"
	// "k8s.io/kubernetes/test/e2e/framework/testfiles"
)

const (
	OpenELBNamespace = "openelb-system"
	OpenELBGrpcPort  = 50051
	OpenELBgpPort    = 17900
	OpenELBgpAS      = 65001
	OpenELBRouterID  = "8.8.8.8"

	PeerGrpcPort  = 50052
	PeerGoBgpPort = 17901
	PeerGoBgpAS   = 65000

	defaultTime = 120
)

func findOpenELBSpeaker(c clientset.Interface, port int) []v1.Pod {
	p, err := c.CoreV1().Pods(OpenELBNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{"app": "openelb", "component": "speaker"}).String(),
	})
	framework.ExpectNoError(err)

	pods := []v1.Pod{}
	klog.Infof("openelb's pod count is %d", len(p.Items))
	for _, pod := range p.Items {
		klog.Infof("get pod for openelb(%s/%s)", OpenELBNamespace, pod.Name)
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", pod.Status.HostIP, port))
		if err != nil {
			klog.Infof("connect error. err :%s", err.Error())
			continue
		}

		conn.Close()
		pods = append(pods, pod)
	}

	return pods
}

var _ = framework.KubesphereDescribe("[OpenELB:BGP]", func() {
	f := framework.NewDefaultFramework("network")

	var c clientset.Interface
	var ns string
	var podClient *framework.PodClient
	var peerPod *v1.Pod
	var peerClient *framework.GobgpClient
	var openelbSpeakerPods []v1.Pod

	ginkgo.BeforeEach(func() {
		c = f.ClientSet
		ns = f.Namespace.Name
		podClient = f.PodClient()

		//get openelb-speaker
		openelbSpeakerPods = findOpenELBSpeaker(c, OpenELBGrpcPort)
		framework.ExpectNotEqual(len(openelbSpeakerPods), 0)
	})

	framework.ConformanceIt("BgpConf", func() {
		ctx := context.Background()

		//setup gobgp openelb peer pod
		ginkgo.By("Starting gobgp peer")
		podName := "gobgp-peer"
		commands := []string{"/usr/local/bin/gobgpd", fmt.Sprintf("--api-hosts=:%d", PeerGrpcPort)}
		pod := framework.MakePod(ns, podName, map[string]string{"app": "gobgp-peer"}, map[string]string{"podsecuritypolicy.policy/disabled": "true"}, "rykren/gobgp:latest", commands, nil)
		_ = podClient.CreateSync(pod)

		err := e2epod.WaitTimeoutForPodReadyInNamespace(c, podName, ns, 30*time.Second)
		framework.ExpectNoError(err)

		peerPod, err = c.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
		framework.ExpectNoError(err)

		peerClient = framework.NewGobgpClient(ctx, peerPod, PeerGrpcPort)
		framework.ExpectNoError(err)
		err = peerClient.AddConfForGobgp(peerPod.Status.PodIP, PeerGoBgpAS, PeerGoBgpPort)
		framework.ExpectNoError(err)

		//config openelb peer info
		for _, p := range openelbSpeakerPods {
			err = peerClient.AddPeerForGobgp(p.Status.HostIP, OpenELBgpAS, OpenELBgpPort)
			framework.ExpectNoError(err)
		}

		// config openelb
		ginkgo.By("Adding Eip")
		eip := &v1alpha2.Eip{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-eip",
			},
			Spec: v1alpha2.EipSpec{
				Address: "192.168.99.10-192.168.99.11",
			},
		}
		framework.ExpectNoError(f.OpenELBClient.Create(ctx, eip))
		defer func() {
			framework.ExpectNoError(f.OpenELBClient.Delete(ctx, eip))
		}()

		ginkgo.By("Adding openelb bgpconf")
		bgpconf := &framework.BgpConfGlobal{
			AS:         OpenELBgpAS,
			ListenPort: int32(OpenELBgpPort),
			Name:       "default",
			Client:     f.OpenELBClient,
			RouterID:   OpenELBRouterID,
		}
		framework.ExpectNoError(bgpconf.Create(ctx))
		defer func() {
			framework.ExpectNoError(bgpconf.Delete(ctx))
		}()

		ginkgo.By("Adding openelb bgppeer")
		bgppeer := &framework.BgpPeer{
			Address: peerPod.Status.PodIP,
			AS:      PeerGoBgpAS,
			Port:    PeerGoBgpPort,
			Name:    "test-peer",
			Client:  f.OpenELBClient,
			Passive: false, //test for passive
		}
		framework.ExpectNoError(bgppeer.Create(ctx))
		defer func() {
			framework.ExpectNoError(bgppeer.Delete(ctx))
		}()

		err = framework.WaitForBGPEstablished(defaultTime*time.Second, peerClient, len(openelbSpeakerPods))
		framework.ExpectNoError(err)

		ginkgo.By("Adding service")
		tcpJig := e2eservice.NewTestJig(c, ns, "test-service")
		_, err = tcpJig.CreateTCPService(nil)
		framework.ExpectNoError(err)
		_, err = tcpJig.UpdateService(func(s *v1.Service) {
			s.Spec.Type = v1.ServiceTypeLoadBalancer
			if s.ObjectMeta.Annotations == nil {
				s.ObjectMeta.Annotations = map[string]string{}
			}

			s.Annotations[constant.OpenELBAnnotationKey] = constant.OpenELBAnnotationValue
			s.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2] = eip.Name
		})

		framework.ExpectNoError(err)
		tcpservice, err := tcpJig.WaitForLoadBalancer(defaultTime * time.Second)
		framework.ExpectNoError(err)
		framework.Logf("ingress %v", tcpservice.Status.LoadBalancer.Ingress)

		//get node count
		ginkgo.By("Getting nodes info")
		nodes, err := c.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		framework.ExpectNoError(err)
		framework.ExpectNotNil(nodes)

		//check router count
		ginkgo.By("Checking router count")
		framework.ExpectNoError(framework.WaitForRouterNum(defaultTime*time.Second, tcpservice.Status.LoadBalancer.Ingress[0].IP, peerClient, len(nodes.Items)))
	})
})
