package cidr

import (
	"bytes"
	"net"
	"strconv"
)

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cidr", func() {
	It("Should be ok to test Subnet", func() {
		type Case struct {
			Base   string
			Bits   int
			Num    int
			Output string
			Error  bool
		}

		cases := []Case{
			Case{
				Base:   "192.168.2.0/20",
				Bits:   4,
				Num:    6,
				Output: "192.168.6.0/24",
			},
			Case{
				Base:   "192.168.2.0/20",
				Bits:   4,
				Num:    0,
				Output: "192.168.0.0/24",
			},
			Case{
				Base:   "192.168.0.0/31",
				Bits:   1,
				Num:    1,
				Output: "192.168.0.1/32",
			},
			Case{
				Base:   "192.168.0.0/21",
				Bits:   4,
				Num:    7,
				Output: "192.168.3.128/25",
			},
			Case{
				Base:   "fe80::/48",
				Bits:   16,
				Num:    6,
				Output: "fe80:0:0:6::/64",
			},
			Case{
				Base:   "fe80::/49",
				Bits:   16,
				Num:    7,
				Output: "fe80:0:0:3:8000::/65",
			},
			Case{
				Base:  "192.168.2.0/31",
				Bits:  2,
				Num:   0,
				Error: true, // not enough bits to expand into
			},
			Case{
				Base:  "fe80::/126",
				Bits:  4,
				Num:   0,
				Error: true, // not enough bits to expand into
			},
			Case{
				Base:  "192.168.2.0/24",
				Bits:  4,
				Num:   16,
				Error: true, // can't fit 16 into 4 bits
			},
		}

		for _, testCase := range cases {
			_, base, _ := net.ParseCIDR(testCase.Base)
			gotNet, err := Subnet(base, testCase.Bits, testCase.Num)
			if testCase.Error {
				Expect(err).Should(HaveOccurred())
			} else {
				Expect(err).ShouldNot(HaveOccurred())
				Expect(gotNet.String()).To(Equal(testCase.Output))
			}
		}
	})
	It("Should be ok to test Host", func() {
		type Case struct {
			Range  string
			Num    int
			Output string
			Error  bool
		}

		cases := []Case{
			Case{
				Range:  "192.168.2.0/20",
				Num:    6,
				Output: "192.168.0.6",
			},
			Case{
				Range:  "192.168.0.0/20",
				Num:    257,
				Output: "192.168.1.1",
			},
			Case{
				Range:  "2001:db8::/32",
				Num:    1,
				Output: "2001:db8::1",
			},
			Case{
				Range: "192.168.1.0/24",
				Num:   256,
				Error: true, // only 0-255 will fit in 8 bits
			},
			Case{
				Range:  "192.168.0.0/30",
				Num:    -3,
				Output: "192.168.0.1", // 4 address (0-3) in 2 bits; 3rd from end = 1
			},
			Case{
				Range:  "192.168.0.0/30",
				Num:    -4,
				Output: "192.168.0.0", // 4 address (0-3) in 2 bits; 4th from end = 0
			},
			Case{
				Range: "192.168.0.0/30",
				Num:   -5,
				Error: true, // 4 address (0-3) in 2 bits; cannot accomodate 5
			},
			Case{
				Range:  "fd9d:bc11:4020::/64",
				Num:    2,
				Output: "fd9d:bc11:4020::2",
			},
			Case{
				Range:  "fd9d:bc11:4020::/64",
				Num:    -2,
				Output: "fd9d:bc11:4020:0:ffff:ffff:ffff:fffe",
			},
		}

		for _, testCase := range cases {
			_, network, _ := net.ParseCIDR(testCase.Range)
			gotIP, err := Host(network, testCase.Num)
			if testCase.Error {
				Expect(err).Should(HaveOccurred())
			} else {
				Expect(err).ShouldNot(HaveOccurred())
				Expect(gotIP.String()).To(Equal(testCase.Output))
			}
		}
	})

	It("Should be ok to test AddressRange", func() {
		type Case struct {
			Range string
			First string
			Last  string
		}

		cases := []Case{
			Case{
				Range: "192.168.0.0/16",
				First: "192.168.0.0",
				Last:  "192.168.255.255",
			},
			Case{
				Range: "192.168.0.0/17",
				First: "192.168.0.0",
				Last:  "192.168.127.255",
			},
			Case{
				Range: "fe80::/64",
				First: "fe80::",
				Last:  "fe80::ffff:ffff:ffff:ffff",
			},
		}

		for _, testCase := range cases {
			_, network, _ := net.ParseCIDR(testCase.Range)
			firstIP, lastIP := AddressRange(network)
			Expect(firstIP.String()).To(Equal(testCase.First))
			Expect(lastIP.String()).To(Equal(testCase.Last))
		}
	})
	It("Should be ok to test IncDes", func() {

		testCase := [][]string{
			[]string{"0.0.0.0", "0.0.0.1"},
			[]string{"10.0.0.0", "10.0.0.1"},
			[]string{"9.255.255.255", "10.0.0.0"},
			[]string{"255.255.255.255", "0.0.0.0"},
			[]string{"::", "::1"},
			[]string{"ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff", "::"},
			[]string{"2001:db8:c001:ba00::", "2001:db8:c001:ba00::1"},
		}

		for _, tc := range testCase {
			ip1 := net.ParseIP(tc[0])
			ip2 := net.ParseIP(tc[1])
			iIP := Inc(ip1)
			Expect(iIP.Equal(ip2)).To(BeTrue())
			Expect(ip1.Equal(ip2)).To(BeFalse())
		}
		for _, tc := range testCase {
			ip1 := net.ParseIP(tc[0])
			ip2 := net.ParseIP(tc[1])
			dIP := Dec(ip2)
			Expect(ip1.Equal(dIP)).To(BeTrue())
			Expect(ip2.Equal(dIP)).To(BeFalse())
		}
	})
	It("Should be ok to test PreviousSubnet", func() {

		testCases := [][]string{
			[]string{"10.0.0.0/24", "9.255.255.0/24", "false"},
			[]string{"100.0.0.0/26", "99.255.255.192/26", "false"},
			[]string{"0.0.0.0/26", "255.255.255.192/26", "true"},
			[]string{"2001:db8:e000::/36", "2001:db8:d000::/36", "false"},
			[]string{"::/64", "ffff:ffff:ffff:ffff::/64", "true"},
		}
		for _, tc := range testCases {
			_, c1, _ := net.ParseCIDR(tc[0])
			_, c2, _ := net.ParseCIDR(tc[1])
			mask, _ := c1.Mask.Size()
			p1, rollback := PreviousSubnet(c1, mask)
			Expect(p1.IP.Equal(c2.IP)).To(BeTrue(), "IP expected %v, got %v\n", c2.IP, p1.IP)
			Expect(bytes.Equal(p1.Mask, c2.Mask)).To(BeTrue(), "Mask expected %v, got %v\n", c2.Mask, p1.Mask)
			Expect(p1.String()).To(Equal(c2.String()))
			check, _ := strconv.ParseBool(tc[2])
			Expect(rollback).To(Equal(check))
		}
		for _, tc := range testCases {
			_, c1, _ := net.ParseCIDR(tc[0])
			_, c2, _ := net.ParseCIDR(tc[1])
			mask, _ := c1.Mask.Size()
			n1, rollover := NextSubnet(c2, mask)
			Expect(n1.IP.Equal(c1.IP)).To(BeTrue(), "IP expected %v, got %v\n", c1.IP, n1.IP)
			Expect(bytes.Equal(n1.Mask, c1.Mask)).To(BeTrue(), "Mask expected %v, got %v\n", c1.Mask, n1.Mask)
			Expect(n1.String()).To(Equal(c1.String()))
			check, _ := strconv.ParseBool(tc[2])
			Expect(rollover).To(Equal(check))
		}
	})
	It("Should be ok to test VerifyNetowrk", func() {

		type testVerifyNetwork struct {
			CIDRBlock string
			CIDRList  []string
		}

		testCases := []*testVerifyNetwork{
			&testVerifyNetwork{
				CIDRBlock: "192.168.8.0/21",
				CIDRList: []string{
					"192.168.8.0/24",
					"192.168.9.0/24",
					"192.168.10.0/24",
					"192.168.11.0/25",
					"192.168.11.128/25",
					"192.168.12.0/25",
					"192.168.12.128/26",
					"192.168.12.192/26",
					"192.168.13.0/26",
					"192.168.13.64/27",
					"192.168.13.96/27",
					"192.168.13.128/27",
				},
			},
		}
		failCases := []*testVerifyNetwork{
			&testVerifyNetwork{
				CIDRBlock: "192.168.8.0/21",
				CIDRList: []string{
					"192.168.8.0/24",
					"192.168.9.0/24",
					"192.168.10.0/24",
					"192.168.11.0/25",
					"192.168.11.128/25",
					"192.168.12.0/25",
					"192.168.12.64/26",
					"192.168.12.128/26",
				},
			},
			&testVerifyNetwork{
				CIDRBlock: "192.168.8.0/21",
				CIDRList: []string{
					"192.168.7.0/24",
					"192.168.9.0/24",
					"192.168.10.0/24",
					"192.168.11.0/25",
					"192.168.11.128/25",
					"192.168.12.0/25",
					"192.168.12.64/26",
					"192.168.12.128/26",
				},
			},
			&testVerifyNetwork{
				CIDRBlock: "10.42.0.0/24",
				CIDRList: []string{

					"10.42.0.16/28",
					"10.42.0.32/28",
					"10.42.0.0/24",
				},
			},
		}

		for _, tc := range testCases {
			subnets := make([]*net.IPNet, len(tc.CIDRList))
			for i, s := range tc.CIDRList {
				_, n, _ := net.ParseCIDR(s)
				subnets[i] = n
			}
			_, CIDRBlock, _ := net.ParseCIDR(tc.CIDRBlock)
			Expect(VerifyNoOverlap(subnets, CIDRBlock)).Should(BeNil())
		}
		for _, tc := range failCases {
			subnets := make([]*net.IPNet, len(tc.CIDRList))
			for i, s := range tc.CIDRList {
				_, n, _ := net.ParseCIDR(s)
				subnets[i] = n
			}
			_, CIDRBlock, _ := net.ParseCIDR(tc.CIDRBlock)
			Expect(VerifyNoOverlap(subnets, CIDRBlock)).ShouldNot(BeNil(), "Test should have failed with CIDR %s\n", tc.CIDRBlock)
		}
	})
})
