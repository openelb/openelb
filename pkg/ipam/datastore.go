package ipam

import (
	"fmt"
	"github.com/kubesphere/porter/api/v1alpha1"
	bgpserver "github.com/kubesphere/porter/pkg/bgp/serverd"
	"github.com/kubesphere/porter/pkg/constant"
	"github.com/kubesphere/porter/pkg/layer2"
	"net"
	"sync"

	"github.com/go-logr/logr"
	"github.com/kubesphere/porter/pkg/errors"
	"github.com/kubesphere/porter/pkg/util"
	"github.com/mikioh/ipaddr"
	"k8s.io/apimachinery/pkg/types"
)

func NewDataStore(log logr.Logger, bgpServer bgpserver.AnnounceBgp) *DataStore {
	return &DataStore{
		log:        log.WithName("DataStore"),
		IPPool:     make(map[string]*CIDRResource),
		responders: make(map[string]layer2.Responder),
		bgpServer:  bgpServer,
	}
}

type DataStore struct {
	log        logr.Logger
	lock       sync.Mutex
	IPPool     map[string]*CIDRResource
	bgpServer  bgpserver.AnnounceBgp
	responders map[string]layer2.Responder
}

//unsafe need locked
func (d *DataStore) isInteractOfCurrentPool(ipnets []*net.IPNet) bool {

	if ipnets == nil {
		return false
	}

	for _, cidr := range d.IPPool {
		if cidr.IntersectsWith(ipnets) {
			return true
		}
	}
	return false
}

type CIDRResource struct {
	EIPRefName    string
	CIDRs         []*net.IPNet
	Used          map[string]*EIPRef
	Size          int
	UsingKnownIPs bool
	Protocol      string
}

func (c *CIDRResource) IsFull() bool {
	return len(c.Used) == c.Size
}

func (c *CIDRResource) Contains(ip net.IP) bool {
	if ip == nil {
		return false
	}

	for _, cidr := range c.CIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func (c *CIDRResource) IntersectsWith(ipnets []*net.IPNet) bool {
	if ipnets == nil {
		return false
	}

	for _, cidr := range c.CIDRs {
		for _, ipnet := range ipnets {
			if util.Intersect(cidr, ipnet) {
				return true
			}
		}
	}
	return false
}

type EIPRef struct {
	EIPRefName string
	Address    string
	Service    types.NamespacedName
}

type AssignIPResponse struct {
	EIPRefName string
	Address    string
}

func (d *DataStore) AddEIPPool(eip string, name string, usingKnownIPs bool, protocol string) error {
	log := d.log.WithValues("CIDR", eip)

	if protocol == "" {
		protocol = constant.PorterProtocolBGP
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	ipnets, err := util.ParseAddress(eip)
	if err != nil {
		log.Info("eip is invalid")
		return nil
	}

	if len(ipnets) <= 0 {
		log.Info("EIP range is invalid")
		return nil
	}
	if _, ok := d.IPPool[name]; ok {
		log.Info("Cannot add eips with same name", "name", name)
		return nil
	}
	if d.isInteractOfCurrentPool(ipnets) {
		log.Info("eip overlap")
		return nil
	}

	if protocol == constant.PorterProtocolLayer2 {
		responder, err := layer2.NewResponder(ipaddr.NewCursor([]ipaddr.Prefix{*ipaddr.NewPrefix(ipnets[0])}).First().IP, d.log.WithName(name))
		if err != nil {
			return err
		}
		d.responders[name] = responder
	}

	d.IPPool[name] = &CIDRResource{
		EIPRefName:    name,
		CIDRs:         ipnets,
		Used:          make(map[string]*EIPRef),
		Size:          util.GetValidAddressCount(eip),
		UsingKnownIPs: usingKnownIPs,
		Protocol:      protocol,
	}
	log.Info("Added EIP to pool")
	return nil
}

func (d *DataStore) RemoveEIPPool(eip, name string) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	log := d.log.WithValues("name", name, "eip", eip)

	if responder, ok := d.responders[name]; ok {
		if err := responder.Close(); err != nil {
			log.Error(err, "Cannot close responder")
			return err
		}
		delete(d.responders, name)
	}

	if res, ok := d.IPPool[name]; ok {
		if len(res.Used) != 0 {
			for key, val := range res.Used {
				d.log.Info("Service is still using this pool", "service", val.Service, "ip", key)
			}
			log.Info("DataStore EIP inuse")
			return errors.PorterError{Code: errors.EIPIsUsedError}
		}

		delete(d.IPPool, name)
		return nil
	}
	return nil
}

func validateProtocol(protocol string) (string, error) {
	switch protocol {
	case constant.PorterProtocolBGP:
		fallthrough
	case constant.PorterProtocolLayer2:
		return protocol, nil
	case "":
		return constant.PorterProtocolBGP, nil
	default:
		return "", errors.PorterError{Code: errors.ParaInvalidError}
	}
}

func (d *DataStore) AssignIP(serviceName, ns string, protocol string) (*AssignIPResponse, error) {
	d.log.Info("Try to AssignIP to service", "Service", serviceName, "Namespace", ns)

	protocol, err := validateProtocol(protocol)
	if err != nil {
		return nil, err
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	selectIP := &AssignIPResponse{}
	for _, ips := range d.IPPool {
		if ips.IsFull() || ips.Protocol != protocol {
			continue
		}

		for _, cidr := range ips.CIDRs {
			c := ipaddr.NewCursor([]ipaddr.Prefix{*ipaddr.NewPrefix(cidr)})
			for pos := c.First(); pos != nil; pos = c.Next() {
				ip := pos.IP
				if !ips.UsingKnownIPs {
					last := ip.To4()[3]
					if last == 0 || last == 255 {
						continue
					}
				}
				if _, ok := ips.Used[ip.String()]; !ok {
					ips.Used[ip.String()] = &EIPRef{
						EIPRefName: ips.EIPRefName,
						Address:    ip.String(),
						Service:    types.NamespacedName{Name: serviceName, Namespace: ns},
					}
					selectIP.Address = ip.String()
					selectIP.EIPRefName = ips.EIPRefName
					d.log.Info("Assign IP to service", "Service", serviceName, "ip", selectIP.Address)
					return selectIP, nil
				}
			}
		}
	}
	return nil, errors.PorterError{Code: errors.EIPNotEnoughError}
}

func (d *DataStore) AssignSpecifyIP(ipstr, protocol, serviceName, ns string) (*AssignIPResponse, error) {
	protocol, err := validateProtocol(protocol)
	if err != nil {
		return nil, err
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	ip := net.ParseIP(ipstr)
	for _, ips := range d.IPPool {
		if ips.Contains(ip) {
			if !ips.UsingKnownIPs {
				last := ip.To4()[3]
				if last == 0 || last == 255 {
					return nil, errors.PorterError{Code: errors.EIPNotExist}
				}
			}
			if _, ok := ips.Used[ipstr]; ok {
				return nil, errors.PorterError{Code: errors.EIPIsUsedError}
			}

			ips.Used[ipstr] = &EIPRef{
				EIPRefName: ips.EIPRefName,
				Address:    ipstr,
				Service:    types.NamespacedName{Name: serviceName, Namespace: ns},
			}
			return &AssignIPResponse{
				EIPRefName: ips.EIPRefName,
				Address:    ipstr,
			}, nil
		}
	}
	return nil, errors.PorterError{Code: errors.EIPNotExist}
}

func (d *DataStore) UnassignIP(ipstr string) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	ip := net.ParseIP(ipstr)
	for _, ips := range d.IPPool {
		if ips.Contains(ip) {
			if _, ok := ips.Used[ipstr]; ok {
				delete(ips.Used, ipstr)
				return nil
			}
			return nil
		}
	}
	return nil
}

func (d *DataStore) GetPoolUsage(name string) (v1alpha1.EipStatus, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	status := v1alpha1.EipStatus{}

	pool, ok := d.IPPool[name]
	if !ok {
		return status, fmt.Errorf("EIP %s not exist", name)
	}

	status.PoolSize = pool.Size
	status.Usage = len(pool.Used)
	if status.Usage == status.PoolSize {
		status.Occupied = true
	} else {
		status.Occupied = false
	}

	return status, nil
}

type EIPStatus struct {
	EIPRef *EIPRef
	Used   bool
	Exist  bool
}

func (d *DataStore) GetEIPStatus(eip string) *EIPStatus {
	d.lock.Lock()
	defer d.lock.Unlock()

	ip := net.ParseIP(eip)
	for _, ips := range d.IPPool {
		if ips.Contains(ip) {
			if ref, ok := ips.Used[eip]; ok {
				return &EIPStatus{
					EIPRef: ref,
					Used:   true,
					Exist:  true,
				}
			}
			return &EIPStatus{
				EIPRef: &EIPRef{
					EIPRefName: ips.EIPRefName,
					Address:    eip,
				},
				Used:  false,
				Exist: true,
			}
		}
	}
	return &EIPStatus{}
}
