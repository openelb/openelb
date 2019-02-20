package test

import (
	"github.com/kubesphere/porter/pkg/bgp/routes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BGP routes test", func() {
	Context("Reconcile Routes", func() {
		It("Should generate right number", func() {
			a := routes.GenerateIdentifier("192.168.98.1")
			b := routes.GenerateIdentifier("192.168.98.11")
			c := routes.GenerateIdentifier("192.168.98.133")
			Expect(a).To(BeEquivalentTo(1))
			Expect(b).To(BeEquivalentTo(11))
			Expect(c).To(BeEquivalentTo(133))
		})
		It("Should correctly add/delete all routes", func() {
			ip := "100.100.100.100"
			nexthops := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}
			add, delete, err := routes.ReconcileRoutes(ip, nexthops)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(add).Should(HaveLen(3))
			Expect(delete).Should(HaveLen(0))
			err = routes.AddMultiRoutes(ip, 32, nexthops)
			Expect(err).ShouldNot(HaveOccurred())
			add, delete, err = routes.ReconcileRoutes(ip, nexthops)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(add).Should(HaveLen(0))
			Expect(delete).Should(HaveLen(0))
		})
	})
})
