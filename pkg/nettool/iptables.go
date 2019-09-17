package nettool

import (
	"fmt"
	"strconv"

	"github.com/kubesphere/porter/pkg/nettool/iptables"
)

const BGPPort = "179"

func GenerateCretiriaAndAction(routerIP, localIP string, gobgpPort int32) []string {
	return []string{"-s", routerIP, "-p", "tcp", "--dport", BGPPort, "-j", "DNAT", "--to-destination", fmt.Sprintf("%s:%d", localIP, gobgpPort)}
}

//Example: iptables -t nat -A PREROUTING -s 10.10.12.1 -p tcp --dport 179 -j DNAT --to-destination 10.10.12.1:17900
//example : iptables -t nat  -A POSTROUTING   -d 192.168.98.2 -p tcp --dport 17900 -j  MASQUERADE
func AddPortForwardOfBGP(iptableExec iptables.IptablesIface, routerIP, localIP string, gobgpPort int32) error {
	//SNAT
	rule := GenerateCretiriaAndAction(routerIP, localIP, gobgpPort)
	ok, err := iptableExec.Exists("nat", "PREROUTING", rule...)
	if err != nil {
		return err
	}
	if !ok {
		err = iptableExec.Append("nat", "PREROUTING", rule...)
		if err != nil {
			return err
		}
	}
	//MASQUERADE
	rule = []string{"-d", localIP, "-p", "tcp", "--dport", strconv.Itoa(int(gobgpPort)), "-j", "MASQUERADE"}
	ok, err = iptableExec.Exists("nat", "POSTROUTING", rule...)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return iptableExec.Append("nat", "POSTROUTING", rule...)
}

func DeletePortForwardOfBGP(iptableExec iptables.IptablesIface, routerIP, localIP string, gobgpPort int32) error {
	rule := GenerateCretiriaAndAction(routerIP, localIP, gobgpPort)
	return iptableExec.Delete("nat", "PREROUTING", rule...)
}

func OpenForwardForEIP(iptableExec iptables.IptablesIface, eip string) error {
	ruleSpec := []string{"-d", eip, "-j", "ACCEPT"}
	ok, err := iptableExec.Exists("filter", "FORWARD", ruleSpec...)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return iptableExec.Append("filter", "FORWARD", ruleSpec...)
}

func CloseForwardForEIP(iptableExec iptables.IptablesIface, eip string) error {
	ruleSpec := []string{"-d", eip, "-j", "ACCEPT"}
	ok, err := iptableExec.Exists("filter", "FORWARD", ruleSpec...)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return iptableExec.Delete("filter", "FORWARD", ruleSpec...)
}
