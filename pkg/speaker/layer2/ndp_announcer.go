package layer2

import (
	"fmt"
	"github.com/mdlayher/ndp"
	"github.com/openelb/openelb/pkg/metrics"
	"github.com/openelb/openelb/pkg/util/iprange"
	"github.com/vishvananda/netlink"
	"io"
	"k8s.io/klog/v2"
	"net"
	"net/netip"
	"sync"
)

type ndpAnnouncer struct {
	intf  *net.Interface
	conn  *ndp.Conn
	addrs []netlink.Addr

	stopCh   chan struct{}
	lock     sync.RWMutex
	ip2mac   map[string]net.HardwareAddr
	ipranges map[string]iprange.Range
}

func newNDPAnnouncer(ifi *net.Interface) (*ndpAnnouncer, error) {
	conn, _, err := ndp.Listen(ifi, ndp.LinkLocal)
	if err != nil {
		return nil, fmt.Errorf("creating NDP Announcer for %s, err=%v", ifi.Name, err)
	}

	link, _ := netlink.LinkByIndex(ifi.Index)
	addrs, _ := netlink.AddrList(link, netlink.FAMILY_V6)

	ret := &ndpAnnouncer{
		intf:     ifi,
		conn:     conn,
		addrs:    addrs,
		stopCh:   make(chan struct{}),
		ip2mac:   make(map[string]net.HardwareAddr),
		ipranges: make(map[string]iprange.Range),
	}
	return ret, nil
}

func (n *ndpAnnouncer) AddAnnouncedIP(ip net.IP) error {
	svcIP, err := netip.ParseAddr(ip.String())
	if err != nil {
		return fmt.Errorf("parse service ip error : %v", err)
	}

	if err := n.JoinMulticastGroup(svcIP); err != nil {
		return fmt.Errorf(" ip: %s join multicastgroup err", ip)
	}

	if err := n.Gratuitous(svcIP); err != nil {
		return err
	}

	metrics.UpdateGratuitousSentMetrics(ip.String())
	return nil
}

func (n *ndpAnnouncer) DelAnnouncedIP(ip net.IP) error {
	klog.Infof("cancel respone %s's ndp packet", ip)
	n.lock.Lock()
	defer n.lock.Unlock()

	svcIP, err := netip.ParseAddr(ip.String())
	if err != nil {
		return fmt.Errorf("parse service ip error : %v", err)
	}

	if err := n.LeaveMulticastGroup(svcIP); err != nil {
		return fmt.Errorf(" ip: %s leave multicastgroup err", ip)
	}

	delete(n.ip2mac, ip.String())
	metrics.DeleteLayer2Metrics(ip.String())

	return nil
}

func (n *ndpAnnouncer) JoinMulticastGroup(ip netip.Addr) error {
	if !ip.Is6() {
		return fmt.Errorf("join multicastgroup need ipv6")
	}

	multicastGroup, err := ndp.SolicitedNodeMulticast(ip)
	if err != nil {
		return fmt.Errorf("no match solicitedNodeMulticast: %v", err)
	}
	if err = n.conn.JoinGroup(multicastGroup); err != nil {
		return fmt.Errorf("failed to join multicast group: %v", err)
	}

	return nil
}

func (n *ndpAnnouncer) LeaveMulticastGroup(ip netip.Addr) error {
	if !ip.Is6() {
		return fmt.Errorf("leave multicastgroup need ipv6")
	}

	multicastGroup, err := ndp.SolicitedNodeMulticast(ip)
	if err != nil {
		return fmt.Errorf("no match solicitedNodeMulticast: %v", err)
	}
	if err = n.conn.LeaveGroup(multicastGroup); err != nil {
		return fmt.Errorf("failed to leave multicast group: %v", err)
	}

	return nil
}

func (n *ndpAnnouncer) Start() error {
	go func() {
		for {
			select {
			case <-n.stopCh:
				klog.Infof("ndp announcer %s stop Successfully", n.intf.Name)
				return
			default:
				err := n.processRequest()
				if err == dropReasonClosed {
					return
				}
			}
		}
	}()

	return nil
}

func (n *ndpAnnouncer) Stop() error {
	n.conn.Close()
	n.stopCh <- struct{}{}
	return nil
}

func (n *ndpAnnouncer) ContainsIP(ip net.IP) bool {
	for _, r := range n.ipranges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

func (n *ndpAnnouncer) RegisterIPRange(name string, r iprange.Range) {
	n.ipranges[name] = r
}

func (n *ndpAnnouncer) UnregisterIPRange(name string) {
	r, exist := n.ipranges[name]
	if !exist {
		return
	}

	n.lock.RLock()
	defer n.lock.RUnlock()
	for ip, _ := range n.ip2mac {
		if r.Contains(net.ParseIP(ip)) {
			delete(n.ip2mac, ip)
		}
	}

	delete(n.ipranges, name)
}

func (n *ndpAnnouncer) Size() int {
	return len(n.ipranges)
}

func (n *ndpAnnouncer) processRequest() dropReason {
	msg, _, nsSenderIP, err := n.conn.ReadFrom()
	if err != nil {
		if err == io.EOF {
			klog.Infof("ndp speaker interface %s closed", n.intf.Name)
			return dropReasonClosed
		}
		klog.Errorf("ndp speaker read interface %s error: %v", n.intf.Name, err)
		return dropReasonError
	}

	ns, ok := msg.(*ndp.NeighborSolicitation)
	if !ok {
		return dropReasonNotNeighborSolicitation
	}

	var nsSenderHwAddr net.HardwareAddr
	for _, o := range ns.Options {
		nsHwAddr, ok := o.(*ndp.LinkLayerAddress)
		if !ok || nsHwAddr.Direction != ndp.Source {
			continue
		}
		nsSenderHwAddr = nsHwAddr.Addr
		break
	}
	if nsSenderHwAddr == nil {
		return dropReasonNotSourceDirection
	}

	hwAddr := n.getMac(ns.TargetAddress.String())
	if hwAddr == nil {
		return dropReasonUnknowTargetIP
	}

	metrics.UpdateRequestsReceivedMetrics(ns.TargetAddress.String())
	klog.Infof("interface %s got NDP request, send response %s-%s to %s-%s .",
		n.intf.Name, ns.TargetAddress, *hwAddr, nsSenderIP, nsSenderHwAddr)

	na := generateNDP(false, *hwAddr, ns.TargetAddress)
	if err := n.conn.WriteTo(na, nil, nsSenderIP); err != nil {
		klog.Errorf("failed to send neighbor advertisement packet: %v", err)
		return dropReasonError
	}
	metrics.UpdateResponsesSentMetrics(ns.TargetAddress.String())
	return dropReasonNone
}

func (n *ndpAnnouncer) getMac(ip string) *net.HardwareAddr {
	n.lock.RLock()
	defer n.lock.RUnlock()
	result, ok := n.ip2mac[ip]
	if !ok {
		return nil
	}
	return &result
}

func (n *ndpAnnouncer) setMac(ip string, mac net.HardwareAddr) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.ip2mac[ip]; !ok {
		metrics.InitLayer2Metrics(ip)
	}
	n.ip2mac[ip] = mac
}

func generateNDP(gratuitous bool, srcHW net.HardwareAddr, srcIP netip.Addr) *ndp.NeighborAdvertisement {
	na := &ndp.NeighborAdvertisement{
		Router:        false,
		Solicited:     !gratuitous,
		Override:      gratuitous,
		TargetAddress: srcIP,
		Options: []ndp.Option{
			&ndp.LinkLayerAddress{
				Direction: ndp.Target,
				Addr:      srcHW,
			},
		},
	}

	return na
}

func (n *ndpAnnouncer) Gratuitous(ip netip.Addr) error {
	if n.getMac(ip.String()) != nil {
		return nil
	}

	n.setMac(ip.String(), n.intf.HardwareAddr)
	klog.Infof("store ingress ip related node ip and mac. %s-%s", ip.String(), n.intf.HardwareAddr.String())

	addr, err := netip.ParseAddr(net.IPv6linklocalallnodes.String())
	if err != nil {
		return fmt.Errorf("parse IPv6linklocalallnodes: %v", err)
	}

	klog.Infof("send gratuitous ndp packet: %s-%s", ip, n.intf.HardwareAddr)
	na := generateNDP(true, n.intf.HardwareAddr, ip)
	return n.conn.WriteTo(na, nil, addr)
}
