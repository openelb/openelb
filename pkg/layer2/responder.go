package layer2

import (
	"fmt"
	"net"

	"github.com/go-logr/logr"
	"github.com/vishvananda/netlink"
)

type Responder interface {
	DeleteIP(ip string)
	Gratuitous(ip, nodeIP net.IP) error
	Close() error
}

func NewResponder(ip net.IP, log logr.Logger) (Responder, error) {

	routers, err := netlink.RouteGet(ip)
	if err != nil {
		return nil, err
	}

	iface, err := net.InterfaceByIndex(routers[0].LinkIndex)
	if err != nil {
		return nil, err
	}

	if ip.To4() != nil {
		resp, err := newARPResponder(log, iface)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}

	return nil, fmt.Errorf("Not vaild ip, only support ipv4 now")
}

// dropReason is the reason why a layer2 protocol packet was not
// responded to.
type dropReason int

// Various reasons why a packet was dropped.
const (
	dropReasonNone dropReason = iota
	dropReasonClosed
	dropReasonError
	dropReasonARPReply
	dropReasonMessageType
	dropReasonNoSourceLL
	dropReasonEthernetDestination
	dropReasonAnnounceIP
)
