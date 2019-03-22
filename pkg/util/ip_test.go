package util_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/kubesphere/porter/pkg/util"
)

var _ = Describe("Ip", func() {
	It("Should get my default gateway", func() {
		ip := GetOutboundIP()
		Expect(ip).ShouldNot(BeEmpty())
		nodeIP := os.Getenv("NODE_IP")
		if nodeIP != "" {
			Expect(ip).To(Equal(nodeIP))
		}
	})
	It("Should get the default interface", func() {
		name := GetDefaultInterfaceName()
		Expect(name).ShouldNot(BeEmpty())
		intf := os.Getenv("DEFAULT_INTERFACE")
		if intf != "" {
			Expect(name).To(Equal(intf))
		}
	})
})
