package nettool

import (
	"fmt"

	utildbus "k8s.io/kubernetes/pkg/util/dbus"
	"k8s.io/kubernetes/pkg/util/iptables"
	utilexec "k8s.io/utils/exec"
)

const BGPPort = "179"

var IPTablesRunner iptables.Interface

func init() {
	IPTablesRunner = iptables.New(utilexec.New(), utildbus.New(), iptables.ProtocolIpv4)
}

func GenerateCretiriaAndAction(routerIP, localIP string, gobgpPort int32) []string {
	result := []string{}
	//add source
	result = append(result, "-s", routerIP)
	// add protocol
	result = append(result, "-p", "tcp")
	// add BGP dport
	result = append(result, "--dport", BGPPort)
	//  add action
	result = append(result, "-j", "DNAT")
	// add NAT destination
	result = append(result, "--to-destination", fmt.Sprintf("%s:%d", localIP, gobgpPort))
	return result
}

//Example: iptables -t nat -A PREROUTING -s 10.10.12.1 -p tcp --dport 179 -j DNAT --to-destination 10.10.12.1:17900
func AddPortForwardOfBGP(routerIP, localIP string, gobgpPort int32) error {
	rule := GenerateCretiriaAndAction(routerIP, localIP, gobgpPort)
	_, err := IPTablesRunner.EnsureRule(iptables.Append, iptables.TableNAT, iptables.ChainPrerouting, rule...)
	if err != nil {
		return err
	}
	return nil
}

func DeletePortForwardOfBGP(routerIP, localIP string, gobgpPort int32) error {
	rule := GenerateCretiriaAndAction(routerIP, localIP, gobgpPort)
	return IPTablesRunner.DeleteRule(iptables.TableNAT, iptables.ChainPrerouting, rule...)
}
