package layer2

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/go-logr/logr"
	"github.com/j-keck/arping"
	"github.com/mdlayher/arp"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"github.com/openelb/openelb/pkg/metrics"
	"github.com/openelb/openelb/pkg/util/iprange"
	"github.com/vishvananda/netlink"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	protocolARP = 0x0806

	ERROR_NotContainsIP = "node's IP is not in the same network segment as the announced IP."
)

var _ Announcer = &arpAnnouncer{}

type arpAnnouncer struct {
	logger logr.Logger

	intf  *net.Interface
	addrs []netlink.Addr
	conn  *arp.Client
	p     *raw.Conn

	stopCh   chan struct{}
	lock     sync.RWMutex
	ip2mac   map[string]net.HardwareAddr
	ipranges map[string]iprange.Range
}

func (a *arpAnnouncer) RegisterIPRange(name string, r iprange.Range) {
	a.ipranges[name] = r
}

func (a *arpAnnouncer) UnregisterIPRange(name string) {
	r, exist := a.ipranges[name]
	if !exist {
		return
	}

	a.lock.RLock()
	defer a.lock.RUnlock()

	for ip := range a.ip2mac {
		if r.Contains(net.ParseIP(ip)) {
			delete(a.ip2mac, ip)
		}
	}
	delete(a.ipranges, name)
}

func (a *arpAnnouncer) Size() int {
	return len(a.ipranges)
}

func (a *arpAnnouncer) ContainsIP(ip net.IP) bool {
	for _, r := range a.ipranges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

func (a *arpAnnouncer) getMac(ip string) *net.HardwareAddr {
	a.lock.RLock()
	defer a.lock.RUnlock()

	result, ok := a.ip2mac[ip]
	if !ok {
		return nil
	}
	return &result
}

func (a *arpAnnouncer) setMac(ip string, mac net.HardwareAddr) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if _, ok := a.ip2mac[ip]; !ok {
		metrics.InitLayer2Metrics(ip)
	}

	a.ip2mac[ip] = mac
}

func newARPAnnouncer(ifi *net.Interface) (*arpAnnouncer, error) {
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
	ret := &arpAnnouncer{
		logger:   ctrl.Log.WithName("arpSpeaker"),
		intf:     ifi,
		addrs:    addrs,
		conn:     client,
		p:        p,
		stopCh:   make(chan struct{}),
		ip2mac:   make(map[string]net.HardwareAddr),
		ipranges: make(map[string]iprange.Range),
	}

	return ret, nil
}

// The source mac address must be on the network card, otherwise arp spoof could drop you packets.
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

func (a *arpAnnouncer) resolveIP(nodeIP net.IP) (hwAddr net.HardwareAddr, err error) {
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

func (a *arpAnnouncer) gratuitous(ip, nodeIP net.IP) error {
	if a.getMac(ip.String()) != nil {
		return nil
	}

	hwAddr, err := a.resolveIP(nodeIP)
	if err != nil {
		return fmt.Errorf("failed to resolve ip %s, err=%v", nodeIP, err)
	}
	a.setMac(ip.String(), hwAddr)
	a.logger.Info("map ingress ip", "ingress", ip.String(), "nodeIP", nodeIP.String(), "nodeMac", hwAddr.String())

	for _, op := range []arp.Operation{arp.OperationRequest, arp.OperationReply} {
		a.logger.Info("send gratuitous arp packet",
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

func (a *arpAnnouncer) AddAnnouncedIP(ip net.IP) error {
	nexthops := ""
	for _, addr := range a.addrs {
		if addr.Contains(ip) {
			nexthops = addr.IP.String()
		}
	}

	if nexthops == "" {
		return fmt.Errorf("arpAnnouncer add announced IP error : %s", ERROR_NotContainsIP)
	}

	if err := a.gratuitous(ip, net.ParseIP(nexthops)); err != nil {
		return err
	}

	metrics.UpdateGratuitousSentMetrics(ip.String())
	return nil
}

func (a *arpAnnouncer) DelAnnouncedIP(ip net.IP) error {
	klog.Infof("cancel respone %s's arp packet", ip)
	a.lock.Lock()
	defer a.lock.Unlock()

	delete(a.ip2mac, ip.String())
	metrics.DeleteLayer2Metrics(ip.String())

	return nil
}

func (a *arpAnnouncer) Start() error {
	go func() {
		for {
			select {
			case <-a.stopCh:
				klog.Infof("arp announcer %s stop Successfully", a.intf.Name)
				return
			default:
				err := a.processRequest()
				if err == dropReasonClosed {
					return
				}
			}
		}
	}()

	return nil
}

func (a *arpAnnouncer) Stop() error {
	a.conn.Close()
	a.stopCh <- struct{}{}
	return nil
}

func (a *arpAnnouncer) processRequest() dropReason {
	pkt, _, err := a.conn.Read()
	if err != nil {
		if err == io.EOF {
			a.logger.Info("arp speaker closed", "interface", a.intf.Name)
			return dropReasonClosed
		}

		a.logger.Error(err, "arp speaker read error", "interface", a.intf.Name)
		return dropReasonError
	}

	// Ignore ARP replies.
	if pkt.Operation != arp.OperationRequest {
		return dropReasonARPReply
	}

	hwAddr := a.getMac(pkt.TargetIP.String())
	if hwAddr == nil {
		return dropReasonUnknowTargetIP
	}

	metrics.UpdateRequestsReceivedMetrics(pkt.TargetIP.String())
	a.logger.Info("got ARP request, sending response",
		"interface", a.intf.Name,
		"ip", pkt.TargetIP, "senderIP", pkt.SenderIP,
		"senderMAC", pkt.SenderHardwareAddr, "responseMAC", *hwAddr)

	fb, err := generateArp(a.intf.HardwareAddr, arp.OperationReply, *hwAddr, pkt.TargetIP, pkt.SenderHardwareAddr, pkt.SenderIP)
	if err != nil {
		a.logger.Error(err, "generate arp reply packet error")
		return dropReasonError
	}

	if _, err := a.p.WriteTo(fb, &raw.Addr{HardwareAddr: pkt.SenderHardwareAddr}); err != nil {
		a.logger.Error(err, "failed to send response")
		return dropReasonError
	}

	metrics.UpdateResponsesSentMetrics(pkt.TargetIP.String())
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
