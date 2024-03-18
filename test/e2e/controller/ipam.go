package controller

import (
	"context"
	"net"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/test/e2e/framework"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = framework.KubesphereDescribe("[OpenELB:Controller]", func() {
	f := framework.NewDefaultFramework("network")

	var cli client.Client
	var ns string
	var service *corev1.Service
	var externalIP string
	var eipName string

	ginkgo.BeforeEach(func() {
		cli = f.OpenELBClient
		ns = f.Namespace.Name
		eipName = "test-eip-ipam"
	})

	ginkgo.AfterEach(func() {
		framework.ExpectNoError(f.OpenELBClient.Delete(context.Background(), &v1alpha2.Eip{
			ObjectMeta: metav1.ObjectMeta{
				Name: eipName,
			},
		}))
	})

	ginkgo.It("ipam", func() {
		ginkgo.By("Add Eip")
		eip := &v1alpha2.Eip{
			ObjectMeta: metav1.ObjectMeta{
				Name: eipName,
			},
			Spec: v1alpha2.EipSpec{
				Address: "192.168.99.100-192.168.99.101",
			},
		}
		framework.ExpectNoError(f.OpenELBClient.Create(context.Background(), eip))

		ginkgo.By("Creating service")
		service = getPodInfoService(ns, eipName)
		gomega.Expect(cli.Create(context.TODO(), service)).ShouldNot(gomega.HaveOccurred())

		ginkgo.By("Getting service external ip")
		framework.WaitServicePresentFitWith(cli, service.Namespace, service.Name, func(service *corev1.Service) bool {
			return len(service.Status.LoadBalancer.Ingress) != 0
		})

		err := cli.Get(context.TODO(), types.NamespacedName{Namespace: service.Namespace, Name: service.Name}, service)
		framework.ExpectNoError(err)
		externalIP = service.Status.LoadBalancer.Ingress[0].IP

		ginkgo.By("Checking eip status")
		framework.ExpectEqual(eip.Contains(net.ParseIP(externalIP)), true)
		err = f.OpenELBClient.Get(context.TODO(), client.ObjectKey{Namespace: eip.Namespace, Name: eip.Name}, eip)
		framework.ExpectNoError(err)
		framework.ExpectEqual(eip.Status.Usage, 1)
	})

})

func getPodInfoService(ns, eipName string) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podinfo-e2e" + rand.String(3),
			Namespace: ns,
			Annotations: map[string]string{
				constant.OpenELBAnnotationKey:            constant.OpenELBAnnotationValue,
				constant.OpenELBEIPAnnotationKeyV1Alpha2: eipName,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     9898,
					Protocol: corev1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 9898,
					},
				},
			},
			Selector: map[string]string{"app.kubernetes.io/name": "podinfo"},
			Type:     corev1.ServiceTypeLoadBalancer,
		},

		Status: corev1.ServiceStatus{},
	}
	return service
}
