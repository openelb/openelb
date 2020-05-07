package layer2

import (
	"time"

	"github.com/go-logr/logr/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func fakeGetNodeIPMap(c client.Client) (map[string]string, error) {
	return map[string]string{
		"master": testIntfIPStr,
	}, nil
}

var _ = Describe("test announcer", func() {
	It("Set/Unset BanlanceIP", func() {
		getNodeIPMapVar = fakeGetNodeIPMap
		announce := New(testing.NullLogger{}, nil)
		time.Sleep(3 * time.Second)
		resolveIPVar = fakeResolveIP

		Expect(announce.SetBalancer(testEIPStr, testIntfIPStr)).ShouldNot(HaveOccurred())
		Expect(*announce.arp.ip2mac[testEIPStr]).Should(Equal(testIntfHW))

		Expect(announce.DeleteBalancer(testEIPStr)).ShouldNot(HaveOccurred())
		Expect(len(announce.arp.ip2mac)).Should(Equal(0))
	})
})
