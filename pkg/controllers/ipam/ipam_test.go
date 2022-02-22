package ipam

import (
	"fmt"
	"testing"

	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/speaker"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestIpam(t *testing.T) {
	RegisterFailHandler(Fail)
	log := zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter))
	ctrl.SetLogger(log)
	RunSpecs(t, "Ipam Suite")
}

var _ = BeforeSuite(func() {
	speaker.RegisterSpeaker(e.GetSpeakerName(), speaker.NewFake())
	speaker.RegisterSpeaker(e2.GetSpeakerName(), speaker.NewFake())
	IPAMAllocator = &IPAM{
		Client:        nil,
		log:           nil,
		EventRecorder: nil,
	}
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
)

var _ = Describe("IPAMArgs", func() {
	It("should be false, when IPAMResult.Addr is empty", func() {
		a := IPAMArgs{
			Key:      "testsvc",
			Addr:     "",
			Eip:      "",
			Protocol: "",
		}
		r := IPAMResult{
			Addr:     "",
			Eip:      "",
			Protocol: "",
			Sp:       nil,
		}
		Expect(a.ShouldUnAssignIP(r)).ShouldNot(BeTrue())
	})

	It("should be true, when addr not equal", func() {
		a := IPAMArgs{
			Key:      "testsvc",
			Addr:     "192.168.0.1",
			Eip:      "",
			Protocol: "",
		}
		r := IPAMResult{
			Addr:     "192.168.0.2",
			Eip:      "",
			Protocol: "",
			Sp:       nil,
		}
		Expect(a.ShouldUnAssignIP(r)).Should(BeTrue())
	})

	It("should be true, when eip not equal", func() {
		a := IPAMArgs{
			Key:      "testsvc",
			Addr:     "192.168.0.1",
			Eip:      "testeip",
			Protocol: "",
		}
		r := IPAMResult{
			Addr:     "192.168.0.1",
			Eip:      "testeip2",
			Protocol: "",
			Sp:       nil,
		}
		Expect(a.ShouldUnAssignIP(r)).Should(BeTrue())
	})

	It("should be true, when protocol not equal", func() {
		a := IPAMArgs{
			Key:      "testsvc",
			Addr:     "192.168.0.1",
			Eip:      "testeip",
			Protocol: constant.OpenELBProtocolLayer2,
		}
		r := IPAMResult{
			Addr:     "192.168.0.1",
			Eip:      "testeip",
			Protocol: constant.OpenELBProtocolBGP,
			Sp:       nil,
		}
		Expect(a.ShouldUnAssignIP(r)).Should(BeTrue())
	})

	It("should be false, when all fields equal", func() {
		a := IPAMArgs{
			Key:      "testsvc",
			Addr:     "192.168.0.1",
			Eip:      "testeip",
			Protocol: constant.OpenELBProtocolBGP,
		}
		r := IPAMResult{
			Addr:     "192.168.0.1",
			Eip:      "testeip",
			Protocol: constant.OpenELBProtocolBGP,
			Sp:       nil,
		}
		Expect(a.ShouldUnAssignIP(r)).ShouldNot(BeTrue())
	})
})

var _ = Describe("IPAM", func() {
	It("Add Eip", func() {
		IPAMAllocator.updateEip(&e)
		Expect(e.Status.PoolSize).Should(Equal(256))
		Expect(e.Status.Usage).Should(Equal(0))
		Expect(e.Status.Occupied).Should(Equal(false))
		Expect(e.Status.Ready).Should(Equal(true))
		Expect(e.Status.V4).Should(Equal(true))
		Expect(e.Status.FirstIP).Should(Equal("192.168.1.0"))
		Expect(e.Status.LastIP).Should(Equal("192.168.1.255"))

		IPAMAllocator.updateEip(&e2)
		Expect(e2.Status.PoolSize).Should(Equal(256))
		Expect(e2.Status.Usage).Should(Equal(0))
		Expect(e2.Status.Occupied).Should(Equal(false))
		Expect(e2.Status.Ready).Should(Equal(true))
		Expect(e2.Status.V4).Should(Equal(true))
		Expect(e2.Status.FirstIP).Should(Equal("192.168.2.0"))
		Expect(e2.Status.LastIP).Should(Equal("192.168.2.255"))
	})

	Context("Assign ip", func() {
		Context("eip unusable", func() {
			It("eip not ready", func() {
				e.Status.Ready = false

				addr := IPAMArgs{
					Key:      "testsvc",
					Addr:     "192.168.1.1",
					Eip:      "",
					Protocol: constant.OpenELBProtocolBGP,
				}.assignIPFromEip(&e)
				Expect(addr).Should(Equal(""))

				e.Status.Ready = true
			})

			It("eip not enabled", func() {
				e.Spec.Disable = true

				addr := IPAMArgs{
					Key:      "testsvc",
					Addr:     "192.168.1.1",
					Eip:      "",
					Protocol: constant.OpenELBProtocolBGP,
				}.assignIPFromEip(&e)
				Expect(addr).Should(Equal(""))

				e.Spec.Disable = false
			})
		})

		When("IPAMArgs incorrect", func() {
			It("has no protocol", func() {
				addr := IPAMArgs{
					Key:      "testsvc",
					Addr:     "",
					Eip:      "",
					Protocol: "",
				}.assignIPFromEip(&e)
				Expect(addr).Should(Equal(""))
			})

			It("eip and protocol not match", func() {
				addr := IPAMArgs{
					Key:      "testsvc",
					Addr:     "",
					Eip:      e2.Name,
					Protocol: constant.OpenELBProtocolBGP,
				}.assignIPFromEip(&e2)
				Expect(addr).Should(Equal(""))
			})
		})

		When("IPAMArgs correct", func() {
			It("Specify IP address", func() {
				addr := IPAMArgs{
					Key:      "testsvc255",
					Addr:     "192.168.1.255",
					Eip:      "",
					Protocol: constant.OpenELBProtocolBGP,
				}.assignIPFromEip(&e)
				Expect(addr).Should(Equal("192.168.1.255"))
				Expect(len(e.Status.Used)).Should(Equal(1))
				Expect(e.Status.Occupied).Should(Equal(false))
				Expect(e.Status.Usage).Should(Equal(1))
			})

			It("The address should be the same if it is reassigned using the same key.", func() {
				addr := IPAMArgs{
					Key:      "testsvc255",
					Addr:     "",
					Eip:      "",
					Protocol: constant.OpenELBProtocolBGP,
				}.assignIPFromEip(&e)
				Expect(addr).Should(Equal("192.168.1.255"))
				Expect(len(e.Status.Used)).Should(Equal(1))
				Expect(e.Status.Occupied).Should(Equal(false))
				Expect(e.Status.Usage).Should(Equal(1))
			})

			It("Assigning ip addresses cyclically, knowing that eip is running out.", func() {
				addr := ""
				for i := 0; i < 255; i++ {
					addr = IPAMArgs{
						Key:      fmt.Sprintf("testsvc%d", i),
						Addr:     "",
						Eip:      e.Name,
						Protocol: constant.OpenELBProtocolBGP,
					}.assignIPFromEip(&e)
					Expect(addr).Should(Equal(fmt.Sprintf("192.168.1.%d", i)))
					Expect(len(e.Status.Used)).Should(Equal(i + 2))
					Expect(e.Status.Usage).Should(Equal(i + 2))
				}
				Expect(e.Status.Occupied).Should(Equal(true))

				By("eip is full")
				addr = IPAMArgs{
					Key:      "testsvc256",
					Addr:     "",
					Eip:      "",
					Protocol: constant.OpenELBProtocolBGP,
				}.assignIPFromEip(&e)
				Expect(addr).Should(Equal(""))
			})
		})
	})

	It("unAssign ip", func() {
		addr := ""
		for i := 0; i < 255; i++ {
			addr = IPAMArgs{
				Key: fmt.Sprintf("testsvc%d", i),
			}.unAssignIPFromEip(&e, false)
			Expect(addr).Should(Equal(fmt.Sprintf("192.168.1.%d", i)))
			Expect(len(e.Status.Used)).Should(Equal(256 - i - 1))
			Expect(e.Status.Usage).Should(Equal(256 - i - 1))
			Expect(e.Status.Occupied).Should(Equal(false))
		}
	})
})
