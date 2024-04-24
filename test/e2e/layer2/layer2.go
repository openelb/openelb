package e2e

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/test/e2e/framework"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	e2eservice "k8s.io/kubernetes/test/e2e/framework/service"
)

const (
	OpenELBNamespace = "openelb-system"
	OpenELBSpeaker   = "openelb-speaker"

	defaultTime = 120
)

var _ = framework.KubesphereDescribe("[OpenELB:Layer2]", func() {
	f := framework.NewDefaultFramework("network")

	var c clientset.Interface
	var ns string

	ginkgo.BeforeEach(func() {
		c = f.ClientSet
		ns = f.Namespace.Name

		ginkgo.By("Starting layer2 mode")
		ds, err := c.AppsV1().DaemonSets(OpenELBNamespace).Get(context.TODO(), OpenELBSpeaker, metav1.GetOptions{})
		framework.ExpectNoError(err)
		framework.ExpectNotNil(ds)
		framework.ExpectEqual(len(ds.Spec.Template.Spec.Containers), 1)

		container := ds.Spec.Template.Spec.Containers[0]
		for i, arg := range container.Args {
			if strings.Contains(arg, "enable-layer2") {
				container.Args[i] = "--enable-layer2=true"
			}
		}

		_, err = c.AppsV1().DaemonSets(OpenELBNamespace).Update(context.TODO(), ds, metav1.UpdateOptions{})
		framework.ExpectNoError(err)
	})

	ginkgo.It("layer2", func() {
		ctx := context.Background()

		// config openelb
		ginkgo.By("Adding Eip")
		eip := &v1alpha2.Eip{
			ObjectMeta: metav1.ObjectMeta{
				Name: "layer2-eip",
			},
			Spec: v1alpha2.EipSpec{
				Address:   "172.18.0.201-172.18.0.210",
				Interface: "eth0",
				Protocol:  constant.OpenELBProtocolLayer2,
			},
		}
		framework.ExpectNoError(f.OpenELBClient.Create(ctx, eip))
		defer func() {
			framework.ExpectNoError(f.OpenELBClient.Delete(ctx, eip))
		}()

		ginkgo.By("Adding service")
		tcpJig := e2eservice.NewTestJig(c, ns, "test-service")
		_, err := tcpJig.CreateTCPService(ctx, nil)
		framework.ExpectNoError(err)

		_, err = tcpJig.UpdateService(ctx, func(s *v1.Service) {
			s.Spec.Type = v1.ServiceTypeLoadBalancer
			if s.ObjectMeta.Annotations == nil {
				s.ObjectMeta.Annotations = map[string]string{}
			}

			s.Annotations[constant.OpenELBAnnotationKey] = constant.OpenELBAnnotationValue
			s.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2] = eip.Name
		})

		framework.ExpectNoError(err)
		svc, err := tcpJig.WaitForLoadBalancer(ctx, defaultTime*time.Second)
		framework.ExpectNoError(err)
		framework.Logf("ingress %v", svc.Status.LoadBalancer.Ingress)

		_, err = tcpJig.Run(ctx, tcpJig.AddRCAntiAffinity)
		framework.ExpectNoError(err)

		ingressIP := e2eservice.GetIngressPoint(&svc.Status.LoadBalancer.Ingress[0])
		port := strconv.Itoa(int(svc.Spec.Ports[0].Port))
		hostport := net.JoinHostPort(ingressIP, port)
		address := fmt.Sprintf("http://%s/", hostport)
		framework.ExpectNoError(framework.Do(address))
	})
})
