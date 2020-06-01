package util

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/kubesphere/porter/pkg/constant"
	"github.com/mikioh/ipaddr"
)

// Get preferred outbound ip of this machine
func GetOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func GetDefaultInterfaceName() string {
	ip := GetOutboundIP()
	if ip == "" {
		return ""
	}

	infs, _ := net.Interfaces()
	for _, f := range infs {
		addrs, err := f.Addrs()
		if err != nil {
			log.Fatal(err)
		}
		for _, addr := range addrs {
			if strings.Contains(addr.String(), ip) {
				return f.Name
			}
		}
	}
	return ""
}
func ToCommonString(ip string, prefix uint32) string {
	return fmt.Sprintf("%s/%d", ip, prefix)
}

func ParseAddress(addr string) ([]*net.IPNet, error) {
	if strings.Contains(addr, constant.EipRangeSeparator) {
		r := strings.SplitN(addr, constant.EipRangeSeparator, 2)
		if len(r) != 2 {
			return nil, fmt.Errorf("%s is not a valid address range", addr)
		}
		first := net.ParseIP(strings.TrimSpace(r[0]))
		last := net.ParseIP(strings.TrimSpace(r[1]))
		if first == nil || last == nil {
			return nil, fmt.Errorf("%s is not a valid address range", addr)
		}

		var ret []*net.IPNet
		for _, pfx := range ipaddr.Summarize(first, last) {
			n := &net.IPNet{
				IP:   pfx.IP,
				Mask: pfx.Mask,
			}
			ret = append(ret, n)
		}
		return ret, nil
	}

	if !strings.Contains(addr, "/") {
		addr = addr + "/32"
	}

	_, ipnet, err := net.ParseCIDR(addr)
	if err != nil {
		return nil, err
	}

	return []*net.IPNet{ipnet}, nil
}

func GetCIDRAddressCount(cidr string) int {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		if i := net.ParseIP(cidr); i != nil {
			last := i.To4()[3]
			if last == 255 || last == 0 {
				return 0
			}
			return 1
		}
		return 0
	}
	b, a := ipnet.Mask.Size()
	size := 1 << uint(a-b)
	if b <= 24 {
		return size - 2<<uint(24-b)
	} else {
		temp := make(net.IP, len(ip))
		copy(temp, ip)
		temp = temp.To4()
		temp[3] = 0
		if ipnet.Contains(temp) {
			size--
		}
		temp[3] = 255
		if ipnet.Contains(temp) {
			size--
		}
		return size
	}
}

func GetIPRangeAddressCount(first, last string) int {
	fip := net.ParseIP(first).To4()

	if fip == nil {
		return 0
	}

	lip := net.ParseIP(last).To4()

	if lip == nil {
		return 0
	}

	fn := binary.BigEndian.Uint32(fip)
	ln := binary.BigEndian.Uint32(lip)

	if fn > ln {
		return 0
	}

	pf := (fn & 0xFFFFFF00) >> 8
	pl := (ln | 0x000000FF) >> 8

	pad := int(lip[3]) - int(fip[3]) + 1

	if fip[3] == 0 {
		pad--
	}

	if lip[3] == 255 {
		pad--
	}

	return int(int(pl-pf)*254 + pad)
}

func GetValidAddressCount(addr string) int {
	if strings.Contains(addr, constant.EipRangeSeparator) {
		r := strings.SplitN(addr, constant.EipRangeSeparator, 2)
		if len(r) != 2 {
			return 0
		}
		return GetIPRangeAddressCount(r[0], r[1])
	}

	return GetCIDRAddressCount(addr)
}

func Intersect(n1, n2 *net.IPNet) bool {
	return n2.Contains(n1.IP) || n1.Contains(n2.IP)
}
