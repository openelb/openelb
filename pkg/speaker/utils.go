package speaker

import (
	"fmt"
	"net"
	"strings"

	"github.com/openelb/openelb/pkg/util/iprange"
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

func ValidateInterface(netif *net.Interface, r iprange.Range) error {
	addrs, err := netif.Addrs()
	if err != nil {
		return err
	}

	for _, addr := range addrs {
		ip, cidrnet, err := net.ParseCIDR(addr.String())
		if err != nil {
			return err
		}

		if ip.To4() != nil {
			if cidrnet.Contains(r.Start()) && cidrnet.Contains(r.End()) {
				return nil
			}
		}
		if ip.To16() != nil {
			if cidrnet.Contains(r.Start()) && cidrnet.Contains(r.End()) {
				return nil
			}
		}
	}

	return fmt.Errorf("%s's ip and the eip[%s] are not in the same network segment", netif.Name, r.String())
}
