package util

import (
	"fmt"
	"log"
	"net"
	"os/exec"
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

func ExecIPRuleCommand(operation, eip, table string) error {
	command := "ip rule " + operation + " to " + eip + "/32" + " lookup " + table
	_, err := exec.Command("bash", "-c", command).Output()
	return err
}

func ToCommonString(ip string, prefix uint32) string {
	return fmt.Sprintf("%s/%d", ip, prefix)
}
