package layer2

import (
	"github.com/go-logr/logr/testing"
	portererror "github.com/kubesphere/porter/pkg/errors"
	"github.com/mdlayher/raw"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net"
)

var _ = Describe("test announcer", func() {
	//Need CAP_NET_ADMIN
	//TODO use udp to test instead raw socket
	intf, err := net.InterfaceByName("lo")
	if err != nil {
		return
	}
	_, err = raw.ListenPacket(intf, protocolARP, nil)
	if err != nil {
		return
	}

	It("Set/Unset BanlanceIP", func() {
		announce := New(testing.NullLogger{})

		Expect(announce.SetBalancer(testEIPStr, testIntfIPStr)).Should(Equal(portererror.Layer2AnnouncerNotReadyError{}))

		Expect(announce.DeleteBalancer(testEIPStr)).ShouldNot(HaveOccurred())
	})
})
