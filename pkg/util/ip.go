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
