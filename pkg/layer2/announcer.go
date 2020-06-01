package layer2

import (
	"github.com/go-logr/logr"
	portererror "github.com/kubesphere/porter/pkg/errors"
	"github.com/vishvananda/netlink"
	"net"
)

// Announce is used to "announce" new IPs mapped to the node's MAC address.
type Announce struct {
	logger     logr.Logger
	responders map[string]Responder
}

// New returns an initialized Announce.
func New(log logr.Logger) *Announce {
	ret := &Announce{
		logger:     log.WithName("Announcer"),
		responders: make(map[string]Responder),
	}
	return ret
}

func (a *Announce) AddResponder(name string, ip net.IP) error {
	if a.responders[name] != nil {
		return nil
	}

	routers, err := netlink.RouteGet(ip)
	if err != nil {
		return err
	}

	iface, err := net.InterfaceByIndex(routers[0].LinkIndex)
	if err != nil {
		return err
	}

	if ip.To4() != nil {
		resp, err := newARPResponder(a.logger, iface)
		if err != nil {
			return err
		}

		a.responders[name] = resp
	}

	return nil
}

func (a *Announce) DeleteResponder(name string) error {
	a.responders[name].close()
	delete(a.responders, name)
	return nil
}

// SetBalancer adds ip to the set of announced addresses.
func (a *Announce) SetBalancer(ip, nodeIP string) error {
	a.logger.Info("set layer2 balancer", "ip", ip, "nodeIP", nodeIP)

	if len(a.responders) <= 0 {
		return portererror.NewLayer2AnnouncerNotReadyError()
	}
	for _, responder := range a.responders {
		if err := responder.gratuitous(net.ParseIP(ip), net.ParseIP(nodeIP)); err != nil {
			return err
		}
	}

	return nil
}

// DeleteBalancer deletes an address from the set of addresses we should announce.
func (a *Announce) DeleteBalancer(ip string) error {
	for _, responder := range a.responders {
		responder.deleteIP(ip)
	}

	return nil
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

type Responder interface {
	deleteIP(ip string)
	gratuitous(ip, nodeIP net.IP) error
	close() error
}
