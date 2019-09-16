package ipam

import (
	"net"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/kubesphere/porter/pkg/errors"
	"github.com/kubesphere/porter/pkg/util"
	"github.com/kubesphere/porter/pkg/util/cidr"
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

func (d *DataStore) IsInteractOfCurrentPool(ipnet *net.IPNet) bool {
	for _, cidr := range d.IPPool {
		if util.Intersect(ipnet, cidr.CIDR) {
			return true
		}
	}
	return false
}

type CIDRResource struct {
	EIPRefName    string
	CIDR          *net.IPNet
	Used          map[string]*EIPRef
	Size          int
	UsingKnownIPs bool
}

func (c *CIDRResource) IsFull() bool {
	return len(c.Used) == c.Size
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
	if !strings.Contains(eip, "/") {
		eip = eip + "/32"
	}
	_, ipnet, err := net.ParseCIDR(eip)
	if err != nil {
		return err
	}
	if _, ok := d.IPPool[name]; ok {
		d.Log.Info("Cannot add eips with same name")
		return errors.DataStoreEIPDuplicateError{CIDR: eip}
	}
	if d.IsInteractOfCurrentPool(ipnet) {
		return errors.DataStoreEIPDuplicateError{CIDR: eip}
	}
	d.lock.Lock()
	defer d.lock.Unlock()
	d.IPPool[name] = &CIDRResource{
		EIPRefName:    name,
		CIDR:          ipnet,
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
	d.Log.Info("Try to AssignIP to service", "Service", serviceName)
	selectIP := &AssignIPResponse{}
	for _, ips := range d.IPPool {
		if ips.IsFull() {
			continue
		}
		cidr.LoopForEachAddressInCIDR(ips.CIDR, func(ip net.IP) bool {
			conti := false
			if !ips.UsingKnownIPs {
				last := ip.To4()[3]
				if last == 0 || last == 255 {
					conti = true
				}
			}
			if conti {
				return false
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
				return true
			}
			return false
		})
	}
	if selectIP.Address == "" {
		return nil, errors.NewResourceNotEnoughError("EIP")
	}
	d.Log.Info("Assign IP to service", "Service", serviceName, "ip", selectIP.Address)
	return selectIP, nil
}

func (d *DataStore) AssignSpecifyIP(ipstr, serviceName, ns string) (*AssignIPResponse, error) {
	ip := net.ParseIP(ipstr)
	for _, ips := range d.IPPool {
		if ips.CIDR.Contains(ip) {
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
		if ips.CIDR.Contains(ip) {
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
		if ips.CIDR.Contains(ip) {
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
