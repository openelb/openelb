package layer2

import (
	"net"

	"github.com/mdlayher/arp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	testIntfHW, _ = net.ParseMAC("00:00:00:00:00:01")
	testIntfHWStr = "00:00:00:00:00:01"
	testIntfIP    = net.ParseIP(testIntfIPStr[0])
	testIntfIPStr = []string{"127.0.0.1"}
	testEIPHW, _  = net.ParseMAC("00:00:00:00:00:02")
	testEIP       = net.ParseIP("192.168.88.2")
	testEIPStr    = "192.168.88.2"
	testDstHW, _  = net.ParseMAC("00:00:00:00:00:03")
	testDstIP     = net.ParseIP("192.168.88.3")
	testDstIPStr  = "192.168.88.3"
)

var _ = Describe("generateArp", func() {
	It("generateArp", func() {
		_, err := generateArp(testIntfHW, arp.OperationReply, testEIPHW, testEIP, testDstHW, testDstIP)
		Expect(err).ShouldNot(HaveOccurred())
	})
})
