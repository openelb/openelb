package nettool

import (
	"net"

	"github.com/kubesphere/porter/pkg/util"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

var link netlink.Link
var AgentTable int

// EIPRoute is a specified ip route which our agent use on start. Equivalent to: "ip route replace local 0/0 dev lo table"
type EIPRoute struct {
}

func init() {
	interfaceName := util.GetDefaultInterfaceName()
	link, _ = netlink.LinkByName(interfaceName)
	AgentTable = 101
}

// EIPRule is a specified ip rule which our agent will use. Equivalent to: "ip rule from all to eip/32 lookup 101"
type EIPRule struct {
	EIP *net.IPNet
}

func NewEIPRule(eip string, mask int) *EIPRule {
	EIP := &net.IPNet{
		IP:   net.ParseIP(eip),
		Mask: net.CIDRMask(mask, 32),
	}
	return &EIPRule{
		EIP: EIP,
	}
}
func (e *EIPRule) ToAgentRule() *netlink.Rule {
	rule := netlink.NewRule()
	src := &net.IPNet{
		IP:   net.IPv4(0, 0, 0, 0),
		Mask: net.IPv4Mask(0, 0, 0, 0),
	}
	rule.Src = src
	rule.Table = AgentTable
	rule.Dst = e.EIP
	return rule
}
func (e *EIPRule) Add() error {
	return netlink.RuleAdd(e.ToAgentRule())
}

func (e *EIPRule) Delete() error {
	return netlink.RuleDel(e.ToAgentRule())
}

func (e *EIPRule) IsExist() (bool, error) {
	rules, err := netlink.RuleList(netlink.FAMILY_V4)
	if err != nil {
		return false, err
	}

	for _, item := range rules {
		if item.Dst != nil && item.Dst.String() == e.EIP.String() {
			return true, nil
		}
	}
	return false, nil
}

//Equivalent to: "ip route replace local 0/0 dev lo table"
func (e *EIPRoute) ToNetlinkRoute() *netlink.Route {
	lo, _ := netlink.LinkByName("lo")
	return &netlink.Route{
		LinkIndex: lo.Attrs().Index,
		Type:      unix.RTN_LOCAL,
		Src:       net.IPv4(172, 0, 0, 1),
		Dst:       nil,
		Table:     AgentTable,
	}
}
func (e *EIPRoute) Add() error {
	return netlink.RouteReplace(e.ToNetlinkRoute())
}

func (e *EIPRoute) Delete() error {
	return netlink.RouteDel(e.ToNetlinkRoute())
}

func (e *EIPRoute) IsExist() (bool, error) {
	routes, err := netlink.RouteList(link, netlink.FAMILY_V4)
	if err != nil {
		return false, err
	}
	r_self := e.ToNetlinkRoute()
	for _, r := range routes {
		if r.Equal(*r_self) {
			return true, nil
		}
	}
	return false, nil
}
