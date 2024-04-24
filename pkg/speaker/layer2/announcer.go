package layer2

import (
	"fmt"
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

func newAnnouncer(iface *net.Interface, v4 bool) (Announcer, error) {
	if v4 {
		return newARPAnnouncer(iface)
	}
	return nil, fmt.Errorf("cannot create layer2 announcer, only support ipv4 now")
}
