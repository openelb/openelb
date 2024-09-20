package speaker

import (
	"fmt"
	"net"
	"strings"

	"github.com/vishvananda/netlink"
)

func ParseInterface(ifaceName string) (iface *net.Interface, err error) {
	strs := strings.SplitN(ifaceName, ":", 2)
	if len(strs) == 1 {
		return net.InterfaceByName(ifaceName)
	}

	switch strs[0] {
	case "can_reach":
		ip := net.ParseIP(strs[1])
		if ip == nil {
			return nil, fmt.Errorf("invalid can_reach address %s", strs[1])
		}

		routers, err := netlink.RouteGet(ip)
		if err != nil {
			return nil, err
		}

		iface, err = net.InterfaceByIndex(routers[0].LinkIndex)
		if err != nil {
			return nil, err
		}

		if iface.Name == "lo" {
			return nil, fmt.Errorf("invalid interface lo")
		}
	default:
		return nil, fmt.Errorf("invalid interface string, now only support can_reach")
	}

	return iface, nil
}
