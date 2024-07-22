package layer2

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/mdlayher/arp"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"github.com/openelb/openelb/pkg/metrics"
	"github.com/openelb/openelb/pkg/util/iprange"
	"github.com/vishvananda/netlink"
	"k8s.io/klog/v2"
)

const (
	protocolARP = 0x0806

	ERROR_NotContainsIP = "node's IP is not in the same network segment as the announced IP."
)

var _ Announcer = &arpAnnouncer{}

type arpAnnouncer struct {
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

func (a *arpAnnouncer) gratuitous(ip net.IP) error {
	if a.getMac(ip.String()) != nil {
		return nil
	}

	a.setMac(ip.String(), a.intf.HardwareAddr)
	klog.Infof("store ingress ip related mac: %s-%s", ip.String(), a.intf.HardwareAddr.String())
	for _, op := range []arp.Operation{arp.OperationRequest, arp.OperationReply} {
		klog.Infof("send gratuitous arp packet: %s-%s", ip, a.intf.HardwareAddr)

		fb, err := generateArp(a.intf.HardwareAddr, op, a.intf.HardwareAddr, ip, ethernet.Broadcast, ip)
		if err != nil {
			klog.Errorf("generate gratuitous arp packet: %v", err)
			return err
		}

		if _, err = a.p.WriteTo(fb, &raw.Addr{HardwareAddr: ethernet.Broadcast}); err != nil {
			klog.Errorf("send gratuitous arp packet: %v", err)
			return err
		}
	}

	return nil
}

func (a *arpAnnouncer) AddAnnouncedIP(ip net.IP) error {
	if err := a.gratuitous(ip); err != nil {
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
			klog.Infof("arp speaker interface %s closed", a.intf.Name)
			return dropReasonClosed
		}

		klog.Errorf("arp speaker read interface %s error: %v", a.intf.Name, err)
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
	klog.Infof("interface %s got ARP request, send response %s-%s to %s-%s .",
		a.intf.Name, pkt.TargetIP, *hwAddr, pkt.SenderIP, pkt.SenderHardwareAddr)
	fb, err := generateArp(a.intf.HardwareAddr, arp.OperationReply, *hwAddr, pkt.TargetIP, pkt.SenderHardwareAddr, pkt.SenderIP)
	if err != nil {
		klog.Errorf("generate arp reply packet error: %v", err)
		return dropReasonError
	}

	if _, err := a.p.WriteTo(fb, &raw.Addr{HardwareAddr: pkt.SenderHardwareAddr}); err != nil {
		klog.Errorf("failed to send response: %v", err)
		return dropReasonError
	}

	metrics.UpdateResponsesSentMetrics(pkt.TargetIP.String())
	return dropReasonNone
}
