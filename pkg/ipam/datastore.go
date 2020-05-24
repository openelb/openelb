package ipam

import (
	"net"
	"sync"

	"github.com/go-logr/logr"
	"github.com/kubesphere/porter/pkg/errors"
	"github.com/kubesphere/porter/pkg/util"
	"github.com/mikioh/ipaddr"
	"k8s.io/apimachinery/pkg/types"
)

func NewDataStore(log logr.Logger) *DataStore {
	return &DataStore{
		Log:    log.WithName("DataStore"),
		IPPool: make(map[string]*CIDRResource),
	}
}

type DataStore struct {
	Log    logr.Logger
	lock   sync.Mutex
	IPPool map[string]*CIDRResource
}

func (d *DataStore) IsInteractOfCurrentPool(ipnets []*net.IPNet) bool {
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

func (d *DataStore) AddEIPPool(eip string, name string, usingKnownIPs bool) error {
	d.Log.Info("Add EIP to pool", "CIDR", eip)

	ipnets, err := util.ParseAddress(eip)

	if err != nil {
		return err
	}

	if _, ok := d.IPPool[name]; ok {
		d.Log.Info("Cannot add eips with same name")
		return errors.DataStoreEIPDuplicateError{CIDR: eip}
	}
	if d.IsInteractOfCurrentPool(ipnets) {
		return errors.DataStoreEIPDuplicateError{CIDR: eip}
	}
	d.lock.Lock()
	defer d.lock.Unlock()

	d.IPPool[name] = &CIDRResource{
		EIPRefName:    name,
		CIDRs:         ipnets,
		Used:          make(map[string]*EIPRef),
		Size:          util.GetValidAddressCount(eip),
		UsingKnownIPs: usingKnownIPs,
	}
	d.Log.Info("Added EIP to pool", "CIDR", eip)
	return nil
}

func (d *DataStore) RemoveEIPPool(eip, name string) error {
	if res, ok := d.IPPool[name]; ok {
		if len(res.Used) != 0 {
			for key, val := range res.Used {
				d.Log.Info("Service is still using this pool", "service", val.Service, "ip", key)
			}
			return errors.DataStoreEIPIsUsedError{CIDR: eip}
		}
		d.lock.Lock()
		defer d.lock.Unlock()
		delete(d.IPPool, name)
		return nil
	}
	return errors.DataStoreEIPNotExist{CIDR: eip}
}

func (d *DataStore) AssignIP(serviceName, ns string) (*AssignIPResponse, error) {
	d.Log.Info("Try to AssignIP to service", "Service", serviceName, "Namespace", ns)
	selectIP := &AssignIPResponse{}
	for _, ips := range d.IPPool {
		if ips.IsFull() {
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
					d.lock.Lock()
					defer d.lock.Unlock()
					ips.Used[ip.String()] = &EIPRef{
						EIPRefName: ips.EIPRefName,
						Address:    ip.String(),
						Service:    types.NamespacedName{Name: serviceName, Namespace: ns},
					}
					selectIP.Address = ip.String()
					selectIP.EIPRefName = ips.EIPRefName
					d.Log.Info("Assign IP to service", "Service", serviceName, "ip", selectIP.Address)
					return selectIP, nil
				}
			}
		}
	}
	return nil, errors.NewResourceNotEnoughError("EIP")
}

func (d *DataStore) AssignSpecifyIP(ipstr, serviceName, ns string) (*AssignIPResponse, error) {
	ip := net.ParseIP(ipstr)
	for _, ips := range d.IPPool {
		if ips.Contains(ip) {
			if !ips.UsingKnownIPs {
				last := ip.To4()[3]
				if last == 0 || last == 255 {
					return nil, errors.DataStoreEIPIsInvalid{EIP: ipstr}
				}
			}
			if _, ok := ips.Used[ipstr]; ok {
				return nil, errors.DataStoreEIPIsUsedError{CIDR: ipstr}
			}
			d.lock.Lock()
			defer d.lock.Unlock()
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
	return nil, errors.NewEIPNotFoundError(ipstr)
}

func (d *DataStore) UnassignIP(ipstr string) error {
	d.Log.Info("Try to UnassignIP", "ip", ipstr)
	ip := net.ParseIP(ipstr)
	for _, ips := range d.IPPool {
		if ips.Contains(ip) {
			if _, ok := ips.Used[ipstr]; ok {
				d.lock.Lock()
				defer d.lock.Unlock()
				delete(ips.Used, ipstr)
				d.Log.Info("UnassignIP done", "ip", ipstr)

				return nil
			}
			return errors.DataStoreEIPIsNotUsedError{EIP: ipstr}
		}
	}
	return errors.NewEIPNotFoundError(ipstr)
}

type EIPStatus struct {
	EIPRef *EIPRef
	Used   bool
	Exist  bool
}

func (d *DataStore) GetEIPStatus(eip string) *EIPStatus {
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
