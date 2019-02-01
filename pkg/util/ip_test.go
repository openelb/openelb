package util_test

import (
	"github.com/kubesphere/porter/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {
	Context("Tests of net function", func() {
		It("Should get right ip", func() {
			ip := util.GetOutboundIP()
			Expect(ip).ShouldNot(BeZero())
		})
		It("Should print right string", func() {
			str := util.ToCommonString("1.0.0.1", 24)
			Expect(str).To(Equal("1.0.0.1/24"))
		})
	})
})
