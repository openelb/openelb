package util

import (
	"fmt"
	"log"
	"net"
	"strings"
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

func GetValidAddressCount(cidr string) int {
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

func Intersect(n1, n2 *net.IPNet) bool {
	return n2.Contains(n1.IP) || n1.Contains(n2.IP)
}
