package layer2

import (
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"

	"github.com/go-logr/logr"
	"github.com/j-keck/arping"
	"github.com/kubesphere/porter/pkg/constant"
	"github.com/kubesphere/porter/pkg/leader-elector"
	"github.com/kubesphere/porter/pkg/speaker"
	"github.com/mdlayher/arp"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"github.com/vishvananda/netlink"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const protocolARP = 0x0806

var _ speaker.Speaker = &arpSpeaker{}

type arpSpeaker struct {
	logger logr.Logger

	intf  *net.Interface
	addrs []netlink.Addr
	conn  *arp.Client
	p     *raw.Conn

	lock   sync.Mutex
	ip2mac map[string]net.HardwareAddr
}

func (a *arpSpeaker) getMac(ip string) *net.HardwareAddr {
	a.lock.Lock()
	defer a.lock.Unlock()

	result, ok := a.ip2mac[ip]
	if !ok {
		return nil
	}
	return &result
}

func (a *arpSpeaker) setMac(ip string, mac net.HardwareAddr) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.ip2mac[ip] = mac
}

func newARPSpeaker(ifi *net.Interface) (*arpSpeaker, error) {
	p, err := raw.ListenPacket(ifi, protocolARP, nil)
	if err != nil {
		return nil, err
	}
	client, err := arp.New(ifi, p)
	if err != nil {
		return nil, fmt.Errorf("creating ARP Speaker for %s, err=%v", ifi.Name, err)
	}

	link, _ := netlink.LinkByIndex(ifi.Index)
	addrs, _ := netlink.AddrList(link, netlink.FAMILY_V4)
	ret := &arpSpeaker{
		logger: ctrl.Log.WithName("arpSpeaker"),
		intf:   ifi,
		addrs:  addrs,
		conn:   client,
		p:      p,
		ip2mac: make(map[string]net.HardwareAddr),
	}

	return ret, nil
}

//The source mac address must be on the network card, otherwise arp spoof could drop you packets.
func generateArp(intfHW net.HardwareAddr, op arp.Operation, srcHW net.HardwareAddr, srcIP net.IP, dstHW net.HardwareAddr, dstIP net.IP) ([]byte, error) {
	pkt, err := arp.NewPacket(op, srcHW, srcIP, dstHW, dstIP)
	if err != nil {
		return nil, err
	}

	pb, err := pkt.MarshalBinary()
	if err != nil {
		return nil, err
	}

	f := &ethernet.Frame{
		Destination: dstHW,
		Source:      intfHW,
		EtherType:   ethernet.EtherTypeARP,
		Payload:     pb,
	}

	fb, err := f.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return fb, err
}

func (a *arpSpeaker) resolveIP(nodeIP net.IP) (hwAddr net.HardwareAddr, err error) {
	routers, err := netlink.RouteGet(nodeIP)
	if err != nil {
		return nil, err
	}

	iface, err := net.InterfaceByIndex(routers[0].LinkIndex)
	if err != nil {
		return nil, err
	}

	if iface.Name == "lo" {
		hwAddr = a.intf.HardwareAddr
	} else {
		//Resolve mac
		for i := 0; i < 3; i++ {
			hwAddr, _, err = arping.PingOverIface(nodeIP, *iface)
			if err != nil {
				hwAddr, _, err = arping.Ping(nodeIP)
				if err != nil {
					continue
				} else {
					break
				}
			} else {
				break
			}
		}
	}

	if hwAddr != nil {
		return hwAddr, nil
	}

	return nil, err
}

func (a *arpSpeaker) gratuitous(ip, nodeIP net.IP) error {
	if !leader.Leader {
		return nil
	}

	hwAddr, err := a.resolveIP(nodeIP)
	if err != nil {
		return fmt.Errorf("failed to resolve ip %s, err=%v", nodeIP, err)
	}

	tmp := a.getMac(ip.String())
	if tmp != nil && reflect.DeepEqual(hwAddr, *tmp) {
		return nil
	}
	a.setMac(ip.String(), hwAddr)

	for _, op := range []arp.Operation{arp.OperationRequest, arp.OperationReply} {
		a.logger.V(4).Info("send gratuitous arp packet",
			"eip", ip, "nodeIP", nodeIP, "hwAddr", hwAddr)

		fb, err := generateArp(a.intf.HardwareAddr, op, hwAddr, ip, ethernet.Broadcast, ip)
		if err != nil {
			a.logger.Error(err, "generate gratuitous arp packet")
			return err
		}

		if _, err = a.p.WriteTo(fb, &raw.Addr{HardwareAddr: ethernet.Broadcast}); err != nil {
			a.logger.Error(err, "send gratuitous arp packet")
			return err
		}
	}

	return nil
}

func (a *arpSpeaker) SetBalancer(ip string, nodes []corev1.Node) error {
	if nodes[0].Annotations != nil {
		nexthop := nodes[0].Annotations[constant.PorterLayer2Annotation]
		if net.ParseIP(nexthop) != nil {
			return a.setBalancer(ip, []string{nexthop})
		}
	}

	for _, addr := range a.addrs {
		for _, tmp := range nodes[0].Status.Addresses {
			if tmp.Type == corev1.NodeInternalIP || tmp.Type == corev1.NodeExternalIP {
				if addr.Contains(net.ParseIP(tmp.Address)) {
					return a.setBalancer(ip, []string{tmp.Address})
				}
			}
		}
	}

	return fmt.Errorf("node %s has no nexthop", nodes[0].Name)
}

func (a *arpSpeaker) setBalancer(ip string, nexthops []string) error {
	return a.gratuitous(net.ParseIP(ip), net.ParseIP(nexthops[0]))
}

func (a *arpSpeaker) DelBalancer(ip string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	delete(a.ip2mac, ip)

	return nil
}

func (a *arpSpeaker) Start(stopCh <-chan struct{}) error {
	go a.run(stopCh)

	go func() {
		<-stopCh
		a.conn.Close()
	}()

	return nil
}

func (a *arpSpeaker) run(stopCh <-chan struct{}) {
	for {
		err := a.processRequest()

		if err == dropReasonClosed {
			return
		} else if err == dropReasonError {
			select {
			case <-stopCh:
				return
			default:
			}
		}
	}
}

func (a *arpSpeaker) processRequest() dropReason {
	pkt, _, err := a.conn.Read()
	if err != nil {
		if err == io.EOF {
			return dropReasonClosed
		}

		return dropReasonError
	}

	if !leader.Leader {
		return dropReasonLeader
	}

	// Ignore ARP replies.
	if pkt.Operation != arp.OperationRequest {
		return dropReasonARPReply
	}

	hwAddr := a.getMac(pkt.TargetIP.String())
	if hwAddr == nil {
		return dropReasonUnknowTargetIP
	}

	a.logger.V(4).Info("got ARP request, sending response",
		"interface", a.intf.Name,
		"ip", pkt.TargetIP, "senderIP", pkt.SenderIP,
		"senderMAC", pkt.SenderHardwareAddr, "responseMAC", *hwAddr)

	fb, err := generateArp(a.intf.HardwareAddr, arp.OperationReply, *hwAddr, pkt.TargetIP, pkt.SenderHardwareAddr, pkt.SenderIP)
	if err != nil {
		a.logger.Error(err, "generate arp reply packet error")
		return dropReasonError
	}

	if _, err := a.p.WriteTo(fb, &raw.Addr{HardwareAddr: pkt.SenderHardwareAddr}); err != nil {
		a.logger.Error(err, "send response",
			"interface", a.intf.Name,
			"ip", pkt.TargetIP, "senderIP", pkt.SenderIP,
			"senderMAC", pkt.SenderHardwareAddr, "responseMAC", *hwAddr)
		return dropReasonError
	}

	return dropReasonNone
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
	dropReasonUnknowTargetIP
	dropReasonLeader
)
