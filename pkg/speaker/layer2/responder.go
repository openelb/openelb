package layer2

import (
	"fmt"
	"net"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"

	"github.com/openelb/openelb/pkg/speaker"
	"github.com/vishvananda/netlink"
)

func NewSpeaker(ifaceName string, v4 bool) (speaker.Speaker, error) {
	var (
		iface *net.Interface
		err   error
	)

	strs := strings.SplitN(ifaceName, ":", 2)
	if len(strs) == 1 {
		iface, err = net.InterfaceByName(ifaceName)
		if err != nil {
			return nil, err
		}
	} else {
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
	}

	ctrl.Log.Info(fmt.Sprintf("use interface %s to speak arp", iface.Name))

	if v4 {
		speaker, err := newARPSpeaker(iface)
		if err != nil {
			return nil, err
		}

		return speaker, nil
	}

	return nil, fmt.Errorf("cannot create layer2 speaker, only support ipv4 now")
}
