package layer2

import (
	"net"

	"github.com/openelb/openelb/pkg/util/iprange"
)

type Announcer interface {
	AddAnnouncedIP(net.IP) error
	DelAnnouncedIP(net.IP) error
	Start() error
	Stop() error
	ContainsIP(net.IP) bool
	RegisterIPRange(string, iprange.Range)
	UnregisterIPRange(string)
	Size() int
}

func newAnnouncer(iface *net.Interface, family iprange.Family) (Announcer, error) {
	if family == iprange.V4Family {
		return newARPAnnouncer(iface)
	}
	return newNDPAnnouncer(iface)
}
