package ipam

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var testEnv *envtest.Environment
var stopCh context.Context
var cancel context.CancelFunc

func TestIpam(t *testing.T) {
	RegisterFailHandler(Fail)
	stopCh, cancel = context.WithCancel(context.Background())
	log := zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter))
	ctrl.SetLogger(log)
	RunSpecs(t, "Ipam Suite")
}

var _ = BeforeSuite(func(done Done) {
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
	}

	cfg, err := testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	// +kubebuilder:scaffold:scheme
	mgr, err := manager.NewManager(cfg, &manager.GenericOptions{
		WebhookPort:   443,
		MetricsAddr:   "0",
		ReadinessAddr: "0",
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(mgr).ToNot(BeNil())

	// Setup all Controllers
	err = SetupIPAM(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err := mgr.Start(stopCh)
		if err != nil {
			ctrl.Log.Error(err, "failed to start manager")
		}
	}()

	// wait other controller, like eip
	time.Sleep(1 * time.Second)
	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var (
	e = v1alpha2.Eip{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "testeip1",
		},
		Spec: v1alpha2.EipSpec{
			Address: "192.168.1.0/24",
		},
		Status: v1alpha2.EipStatus{},
	}

	e2 = v1alpha2.Eip{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "testeip2",
		},
		Spec: v1alpha2.EipSpec{
			Address:   "192.168.2.0/24",
			Protocol:  constant.OpenELBProtocolLayer2,
			Interface: "eth0",
		},
		Status: v1alpha2.EipStatus{},
	}

	svc = &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "testsvc",
			Namespace:   "default",
			Annotations: map[string]string{},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
		Status: corev1.ServiceStatus{},
	}
)

// TODO: add tests
var _ = Describe("IPAMRequest", func() {
	When("Assign ip", func() {

		It("Specify eip and ip", func() {
			clone := svc.DeepCopy()
			clone.Annotations[constant.OpenELBAnnotationKey] = constant.OpenELBAnnotationValue
			clone.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2] = e2.GetName()
			clone.Annotations[constant.OpenELBProtocolAnnotationKey] = e2.GetProtocol()

			i := Request{
				Key: types.NamespacedName{
					Name:      clone.Name,
					Namespace: clone.Namespace,
				}.String(),
			}
			err := i.ConstructAllocate(clone)
			Expect(err).ToNot(HaveOccurred())
			Expect(i.Allocate).ShouldNot(BeNil())

			err = i.ConstructRelease(clone)
			Expect(err).ToNot(HaveOccurred())
			Expect(i.Release).Should(BeNil())
		})

	})

	// When("modify IP", func() {
	// 	Context("modify eip", func() {
	// 		clone := svc.DeepCopy()
	// 		clone.Annotations[constant.OpenELBAnnotationKey] = constant.OpenELBAnnotationValue
	// 		clone.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2] = e.GetName()
	// 		clone.Annotations[constant.OpenELBProtocolAnnotationKey] = e.GetProtocol()

	// 		It("", func() {

	// 		})
	// 		Expect(client.Client).ShouldNot(BeNil())

	// 		err := client.Client.Create(context.Background(), &e)
	// 		Expect(err).ToNot(HaveOccurred())

	// 		err = client.Client.Create(context.Background(), &e2)
	// 		Expect(err).ToNot(HaveOccurred())

	// 		err = client.Client.Create(context.Background(), clone)
	// 		Expect(err).ToNot(HaveOccurred())

	// 		svcGet := &corev1.Service{}
	// 		err = client.Client.Get(context.Background(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, svcGet)
	// 		Expect(err).ToNot(HaveOccurred())

	// 		It("allocated", func() {
	// 			Expect(len(svcGet.Status.LoadBalancer.Ingress)).ShouldNot(Equal(0))
	// 		})

	// 		It("modify", func() {
	// 			svcGet.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2] = e2.GetName()
	// 			svcGet.Annotations[constant.OpenELBProtocolAnnotationKey] = e2.GetProtocol()

	// 			i := IPAMRequest{
	// 				Key: types.NamespacedName{
	// 					Name:      clone.Name,
	// 					Namespace: clone.Namespace,
	// 				}.String(),
	// 			}

	// 			err := i.ConstructAllocate(svcGet)
	// 			Expect(err).ToNot(HaveOccurred())
	// 			Expect(i.Allocate).ShouldNot(BeNil())

	// 			err = i.ConstructRelease(svcGet)
	// 			Expect(err).ToNot(HaveOccurred())
	// 			Expect(i.Release).ShouldNot(BeNil())
	// 		})
	// 	})

	// })

})

// var _ = Describe("IPAM", func() {
// 	It("Add Eip", func() {
// 		IPAMAllocator.updateEip(&e)
// 		Expect(e.Status.PoolSize).Should(Equal(256))
// 		Expect(e.Status.Usage).Should(Equal(0))
// 		Expect(e.Status.Occupied).Should(Equal(false))
// 		Expect(e.Status.Ready).Should(Equal(true))
// 		Expect(e.Status.V4).Should(Equal(true))
// 		Expect(e.Status.FirstIP).Should(Equal("192.168.1.0"))
// 		Expect(e.Status.LastIP).Should(Equal("192.168.1.255"))

// 		IPAMAllocator.updateEip(&e2)
// 		Expect(e2.Status.PoolSize).Should(Equal(256))
// 		Expect(e2.Status.Usage).Should(Equal(0))
// 		Expect(e2.Status.Occupied).Should(Equal(false))
// 		Expect(e2.Status.Ready).Should(Equal(true))
// 		Expect(e2.Status.V4).Should(Equal(true))
// 		Expect(e2.Status.FirstIP).Should(Equal("192.168.2.0"))
// 		Expect(e2.Status.LastIP).Should(Equal("192.168.2.255"))
// 	})

// 	Context("Assign ip", func() {
// 		Context("eip unusable", func() {
// 			It("eip not ready", func() {
// 				e.Status.Ready = false

// 				addr := IPAMArgs{
// 					Key:      "testsvc",
// 					Addr:     "192.168.1.1",
// 					Eip:      "",
// 					Protocol: constant.OpenELBProtocolBGP,
// 				}.assignIPFromEip(&e)
// 				Expect(addr).Should(Equal(""))

// 				e.Status.Ready = true
// 			})

// 			It("eip not enabled", func() {
// 				e.Spec.Disable = true

// 				addr := IPAMArgs{
// 					Key:      "testsvc",
// 					Addr:     "192.168.1.1",
// 					Eip:      "",
// 					Protocol: constant.OpenELBProtocolBGP,
// 				}.assignIPFromEip(&e)
// 				Expect(addr).Should(Equal(""))

// 				e.Spec.Disable = false
// 			})
// 		})

// 		When("IPAMArgs incorrect", func() {
// 			It("has no protocol", func() {
// 				addr := IPAMArgs{
// 					Key:      "testsvc",
// 					Addr:     "",
// 					Eip:      "",
// 					Protocol: "",
// 				}.assignIPFromEip(&e)
// 				Expect(addr).Should(Equal(""))
// 			})

// 			It("eip and protocol not match", func() {
// 				addr := IPAMArgs{
// 					Key:      "testsvc",
// 					Addr:     "",
// 					Eip:      e2.Name,
// 					Protocol: constant.OpenELBProtocolBGP,
// 				}.assignIPFromEip(&e2)
// 				Expect(addr).Should(Equal(""))
// 			})
// 		})

// 		When("IPAMArgs correct", func() {
// 			It("Specify IP address", func() {
// 				addr := IPAMArgs{
// 					Key:      "testsvc255",
// 					Addr:     "192.168.1.255",
// 					Eip:      "",
// 					Protocol: constant.OpenELBProtocolBGP,
// 				}.assignIPFromEip(&e)
// 				Expect(addr).Should(Equal("192.168.1.255"))
// 				Expect(len(e.Status.Used)).Should(Equal(1))
// 				Expect(e.Status.Occupied).Should(Equal(false))
// 				Expect(e.Status.Usage).Should(Equal(1))
// 			})

// 			It("The address should be the same if it is reassigned using the same key.", func() {
// 				addr := IPAMArgs{
// 					Key:      "testsvc255",
// 					Addr:     "",
// 					Eip:      "",
// 					Protocol: constant.OpenELBProtocolBGP,
// 				}.assignIPFromEip(&e)
// 				Expect(addr).Should(Equal("192.168.1.255"))
// 				Expect(len(e.Status.Used)).Should(Equal(1))
// 				Expect(e.Status.Occupied).Should(Equal(false))
// 				Expect(e.Status.Usage).Should(Equal(1))
// 			})

// 			It("Assigning ip addresses cyclically, knowing that eip is running out.", func() {
// 				addr := ""
// 				for i := 0; i < 255; i++ {
// 					addr = IPAMArgs{
// 						Key:      fmt.Sprintf("testsvc%d", i),
// 						Addr:     "",
// 						Eip:      e.Name,
// 						Protocol: constant.OpenELBProtocolBGP,
// 					}.assignIPFromEip(&e)
// 					Expect(addr).Should(Equal(fmt.Sprintf("192.168.1.%d", i)))
// 					Expect(len(e.Status.Used)).Should(Equal(i + 2))
// 					Expect(e.Status.Usage).Should(Equal(i + 2))
// 				}
// 				Expect(e.Status.Occupied).Should(Equal(true))

// 				By("eip is full")
// 				addr = IPAMArgs{
// 					Key:      "testsvc256",
// 					Addr:     "",
// 					Eip:      "",
// 					Protocol: constant.OpenELBProtocolBGP,
// 				}.assignIPFromEip(&e)
// 				Expect(addr).Should(Equal(""))
// 			})
// 		})
// 	})

// 	It("release ip", func() {
// 		addr := ""
// 		for i := 0; i < 255; i++ {
// 			addr = IPAMArgs{
// 				Key: fmt.Sprintf("testsvc%d", i),
// 			}.releaseIPFromEip(&e, false)
// 			Expect(addr).Should(Equal(fmt.Sprintf("192.168.1.%d", i)))
// 			Expect(len(e.Status.Used)).Should(Equal(256 - i - 1))
// 			Expect(e.Status.Usage).Should(Equal(256 - i - 1))
// 			Expect(e.Status.Occupied).Should(Equal(false))
// 		}
// 	})
// })
