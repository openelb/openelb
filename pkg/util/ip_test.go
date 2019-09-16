package util_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/kubesphere/porter/pkg/util"
)

var _ = Describe("Ip test", func() {
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
	It("Should caculate correct ip address", func() {
		Expect(GetValidAddressCount("192.168.1.1")).Should(Equal(1))
		Expect(GetValidAddressCount("192.168.1.0/24")).Should(Equal(254))
		Expect(GetValidAddressCount("192.168.1.1/25")).Should(Equal(127))
		Expect(GetValidAddressCount("192.168.1.0/23")).Should(Equal(508))
		Expect(GetValidAddressCount("192.168.255.255/32")).Should(Equal(0))
		Expect(GetValidAddressCount("192.168.255.255")).Should(Equal(0))
		Expect(GetValidAddressCount("192.168.255.250")).Should(Equal(1))
		Expect(GetValidAddressCount("192.168.255.0")).Should(Equal(0))
		Expect(GetValidAddressCount("192.168.0.0/16")).Should(Equal(65024))
	})
})
