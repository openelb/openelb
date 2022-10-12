package layer2

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/j-keck/arping"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openelb/openelb/pkg/leader-elector"
	"github.com/openelb/openelb/pkg/speaker"
	"github.com/vishvananda/netlink"
	"go.uber.org/zap/zapcore"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	VethIfName     = "testarp1"
	VethIfIP       = "192.168.166.1"
	VethPeerIfName = "testarp2"
	VethPeerIfIP   = "192.168.166.2"
	Gateway        = "192.168.166.254"
	Eip            = "192.168.166.3"
	veth           = &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:  VethIfName,
			Flags: net.FlagUp,
			MTU:   1500,
		},
		PeerName: VethPeerIfName,
	}
)

func TestLayer2(t *testing.T) {
	RegisterFailHandler(Fail)
	if os.Getuid() != 0 {
		//Skip("The test case requires root privileges.")
		return
	}
	log := zap.New(zap.UseDevMode(true), zap.Level(zapcore.DebugLevel), zap.WriteTo(GinkgoWriter))
	ctrl.SetLogger(log)
	RunSpecs(t, "ARP Suite")
}

var _ = BeforeSuite(func() {
	leader.Leader = true

	err := netlink.LinkAdd(veth)
	Expect(err).ShouldNot(HaveOccurred())

	veth1, err := netlink.LinkByName(VethIfName)
	Expect(err).ShouldNot(HaveOccurred())
	err = netlink.LinkSetUp(veth1)
	Expect(err).ShouldNot(HaveOccurred())
	addr1, _ := netlink.ParseAddr("192.168.166.1/24")
	err = netlink.AddrAdd(veth1, addr1)
	Expect(err).ShouldNot(HaveOccurred())

	veth2, err := netlink.LinkByName(VethPeerIfName)
	Expect(err).ShouldNot(HaveOccurred())
	err = netlink.LinkSetUp(veth2)
	Expect(err).ShouldNot(HaveOccurred())
	addr2, _ := netlink.ParseAddr("192.168.166.2/24")
	err = netlink.AddrAdd(veth2, addr2)
	Expect(err).ShouldNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(netlink.LinkDel(veth)).ShouldNot(HaveOccurred())
})

var _ = Describe("new responder", func() {
	Context("invalid interface string", func() {
		It("specify invalid interface name", func() {
			_, err := NewSpeaker(VethIfName+"tesfad", true)
			Expect(err).Should(HaveOccurred())

			_, err = NewSpeaker("haha:hahah", true)
			Expect(err).Should(HaveOccurred())
		})
		It("specify can_reach, but ip invalid", func() {
			_, err := NewSpeaker("can_reach:hahah", true)
			Expect(err).Should(HaveOccurred())
		})
		It("specify can_reach, but interface is lo", func() {
			_, err := NewSpeaker("can_reach:127.0.0.1", true)
			Expect(err).Should(HaveOccurred())
		})
	})

	Context("valid interface string", func() {
		It("specify interface name", func() {
			_, err := NewSpeaker(VethIfName, true)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("specify interface name", func() {
			_, err := NewSpeaker("can_reach:192.168.166.254", true)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})

var _ = Describe("ARP Speak", func() {
	var sp *arpSpeaker

	BeforeEach(func() {
		var err error
		iface, _ := net.InterfaceByName(VethIfName)
		sp, err = newARPSpeaker(iface)
		Expect(err).ShouldNot(HaveOccurred())

		Expect(speaker.RegisterSpeaker(VethIfName, sp)).ShouldNot(HaveOccurred())
	})

	AfterEach(func() {
		speaker.UnRegisterSpeaker(VethIfName)
	})

	It("set loadbalancer", func() {
		By("ip address", func() {
			err := sp.setBalancer("192.168.166.3", []string{"192.168.166.1"})
			Expect(err).ShouldNot(HaveOccurred())
			peer, _ := net.InterfaceByName(VethPeerIfName)
			mac, _, err := arping.PingOverIface(net.ParseIP("192.168.166.3"), *peer)
			Expect(err).ShouldNot(HaveOccurred())
			veth, _ := net.InterfaceByName(VethIfName)
			Expect(mac.String()).Should(Equal(veth.HardwareAddr.String()))
		})
		By("ip range", func() {
			err := sp.setNextHopFromIPRange("192.168.166.3", "192.168.0.0/24")
			Expect(err).ShouldNot(HaveOccurred())
			peer, _ := net.InterfaceByName(VethPeerIfName)
			mac, _, err := arping.PingOverIface(net.ParseIP("192.168.166.3"), *peer)
			Expect(err).ShouldNot(HaveOccurred())
			veth, _ := net.InterfaceByName(VethIfName)
			Expect(mac.String()).Should(Equal(veth.HardwareAddr.String()))
		})
	})

	It("del loadbalancer", func() {
		time.Sleep(10 * time.Second)
		err := sp.DelBalancer("192.168.166.3")
		Expect(err).ShouldNot(HaveOccurred())
		peer, _ := net.InterfaceByName(VethPeerIfName)
		_, _, err = arping.PingOverIface(net.ParseIP("192.168.166.3"), *peer)
		Expect(err).Should(HaveOccurred())
	})
})
