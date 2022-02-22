package nettool_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/openelb/openelb/pkg/nettool"
)

var _ = Describe("Nettool", func() {
	It("Should generate right iptables rule", func() {
		Expect(GenerateCretiriaAndAction("10.10.12.1", "10.10.12.2", 17900)).To(ConsistOf("-s", "10.10.12.1", "-p", "tcp", "--dport", "179", "-j", "DNAT", "--to-destination", "10.10.12.2:17900"))
	})
})
