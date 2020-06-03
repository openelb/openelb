package nettool

import (
	"fmt"
	"github.com/kubesphere/porter/pkg/nettool/iptables"
)

const BGPPort = "179"

func GenerateCretiriaAndAction(routerIP, localIP string, gobgpPort int32) []string {
	return []string{"-s", routerIP, "-p", "tcp", "--dport", BGPPort, "-j", "DNAT", "--to-destination", fmt.Sprintf("%s:%d", localIP, gobgpPort)}
}

//Example: iptables -t nat -A PREROUTING -s 10.10.12.1 -p tcp --dport 179 -j DNAT --to-destination 10.10.12.1:17900
func AddPortForwardOfBGP(iptableExec iptables.IptablesIface, routerIP, localIP string, gobgpPort int32) error {
	rule := GenerateCretiriaAndAction(routerIP, localIP, gobgpPort)
	ok, err := iptableExec.Exists("nat", BgpNatChain, rule...)
	if err != nil {
		return err
	}
	if !ok {
		err = iptableExec.Append("nat", BgpNatChain, rule...)
		if err != nil {
			return err
		}
	}

	return nil
}

func DeletePortForwardOfBGP(iptableExec iptables.IptablesIface, routerIP, localIP string, gobgpPort int32) error {
	rule := GenerateCretiriaAndAction(routerIP, localIP, gobgpPort)
	return iptableExec.Delete("nat", BgpNatChain, rule...)
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

const BgpNatChain = "PREROUTING-PORTER"
