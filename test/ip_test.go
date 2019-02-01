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
			Expect(ip).To(Equal("172.31.129.11"))
		})
	})
})
