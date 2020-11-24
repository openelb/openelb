package v1alpha2

import (
	"net"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func TestTypes(t *testing.T) {
	RegisterFailHandler(Fail)
	log := zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter))
	ctrl.SetLogger(log)
	RunSpecs(t, "v1alpha2 types Suite")
}

var _ = Describe("Test eip types", func() {
	It("Test GetSize", func() {
		e := &Eip{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: EipSpec{
				Address: "192.168.0.1",
			},
			Status: EipStatus{},
		}

		base, size, err := e.GetSize()
		Expect(base.String()).Should(Equal("192.168.0.1"))
		Expect(size).Should(Equal(int64(1)))
		Expect(err).ShouldNot(HaveOccurred())

		e.Spec.Address = "192.168.0.1/24"
		base, size, err = e.GetSize()
		Expect(base.String()).Should(Equal("192.168.0.0"))
		Expect(size).Should(Equal(int64(256)))
		Expect(err).ShouldNot(HaveOccurred())

		e.Spec.Address = "192.168.0.1-192.168.0.100"
		base, size, err = e.GetSize()
		Expect(base.String()).Should(Equal("192.168.0.1"))
		Expect(size).Should(Equal(int64(100)))
		Expect(err).ShouldNot(HaveOccurred())

		e.Spec.Address = "192.168.0.100-192.168.0.1"
		base, size, err = e.GetSize()
		Expect(err).Should(HaveOccurred())

		e.Spec.Address = "xxxx"
		base, size, err = e.GetSize()
		Expect(err).Should(HaveOccurred())

		e.Spec.Address = "192.168.0.1-192.168.0.100-192.168.0.200"
		base, size, err = e.GetSize()
		Expect(err).Should(HaveOccurred())
	})

	It("Test IPToOrdinal", func() {
		e := &Eip{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: EipSpec{
				Address: "192.168.0.1-192.168.0.100",
			},
			Status: EipStatus{},
		}

		offset := e.IPToOrdinal(net.ParseIP("192.168.0.2"))
		Expect(offset).Should(Equal(1))

		offset = e.IPToOrdinal(net.ParseIP("192.168.0.0"))
		Expect(offset).Should(Equal(-1))

		offset = e.IPToOrdinal(net.ParseIP("192.168.0.101"))
		Expect(offset).Should(Equal(-1))
	})

	It("Test IsOverlap", func() {
		e := &Eip{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: EipSpec{
				Address: "192.168.0.100-192.168.0.200",
			},
			Status: EipStatus{},
		}

		e2 := Eip{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: EipSpec{
				Address: "192.168.0.1-192.168.0.99",
			},
			Status: EipStatus{},
		}
		Expect(e.IsOverlap(e2)).Should(BeFalse())

		e2.Spec.Address = "192.168.0.201-192.168.0.250"
		Expect(e.IsOverlap(e2)).Should(BeFalse())

		e2.Spec.Address = "192.168.0.1-192.168.0.100"
		Expect(e.IsOverlap(e2)).Should(BeTrue())

		e2.Spec.Address = "192.168.0.200-192.168.0.250"
		Expect(e.IsOverlap(e2)).Should(BeTrue())
	})

	It("Test ValidateUpdate", func() {
		e := &Eip{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: EipSpec{
				Address: "192.168.0.100-192.168.0.200",
			},
			Status: EipStatus{},
		}

		e2 := e.DeepCopy()
		e2.Spec.Address = "192.168.0.100"
		Expect(e2.ValidateUpdate(e)).Should(HaveOccurred())

		e2 = e.DeepCopy()
		e2.Spec.Disable = true
		Expect(e2.ValidateUpdate(e)).ShouldNot(HaveOccurred())
	})
})
