package test

import (
	"github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/strategy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Strategy Test", func() {
	var eiplist *v1alpha1.EipList
	var service *corev1.Service
	BeforeEach(func() {
		eips := []v1alpha1.Eip{}
		eips = append(eips, v1alpha1.Eip{
			Spec: v1alpha1.EipSpec{
				Address: "1.1.1.1",
			},
		},
			v1alpha1.Eip{
				Spec: v1alpha1.EipSpec{
					Address: "1.1.1.2",
				},
			},
			v1alpha1.Eip{
				Spec: v1alpha1.EipSpec{
					Address: "1.1.1.3",
				},
			})

		eiplist = &v1alpha1.EipList{Items: eips}
		eiplist.Items[0].Status.PortsUsage = make(map[int32]string)
		eiplist.Items[1].Status.PortsUsage = make(map[int32]string)
		eiplist.Items[2].Status.PortsUsage = make(map[int32]string)

		service = &corev1.Service{}
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{Port: 1111})
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{Port: 2222})
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{Port: 3333})
	})
	Context("Default Strategy", func() {
		It("Should choose right ip", func() {
			selector, _ := strategy.GetStrategy(strategy.DefaultStrategy)
			eip, err := selector.Select(service, eiplist)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*eip).To(Equal(eiplist.Items[0]))

			eiplist.Items[0].Status.Occupied = true
			eiplist.Items[1].Status.Occupied = true
			eip, err = selector.Select(service, eiplist)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*eip).To(Equal(eiplist.Items[2]))

			eiplist.Items[2].Status.Occupied = true
			_, err = selector.Select(service, eiplist)
			Expect(err.Error()).To(HavePrefix("No enough EIP resource for allocation"))
		})
	})
	Context("PortBased Strategy", func() {
		It("Should choose right ip", func() {
			selector, _ := strategy.GetStrategy(strategy.PortBasedStrategy)
			eip, err := selector.Select(service, eiplist)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*eip).To(Equal(eiplist.Items[0]))

			eiplist.Items[0].Status.PortsUsage[1111] = "yes"
			eiplist.Items[1].Status.PortsUsage[2222] = "yes"
			eip, err = selector.Select(service, eiplist)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(*eip).To(Equal(eiplist.Items[2]))

			eiplist.Items[2].Status.PortsUsage[3333] = "yes"
			_, err = selector.Select(service, eiplist)
			Expect(err.Error()).To(HavePrefix("No suitable ip has empty ports"))
		})
	})
})
