package ipam

import (
	"encoding/binary"
	"math"
	"net"
	"sync"

	"github.com/go-logr/logr"
	"github.com/kubesphere/porter/pkg/errors"
	"github.com/kubesphere/porter/pkg/util"
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

func (d *DataStore) IsInteractOfCurrentPool(ipnet *net.IPNet, first, last net.IP) bool {
	for _, cidr := range d.IPPool {
		if cidr.IntersectsWith(&CIDRResource{
			CIDR:    ipnet,
			FirstIP: first,
			LastIP:  last,
		}) {
			return true
		}
	}
	return false
}

type CIDRResource struct {
	EIPRefName    string
	CIDR          *net.IPNet
	FirstIP       net.IP
	LastIP        net.IP
	Used          map[string]*EIPRef
	Size          int
	UsingKnownIPs bool
}

func (c *CIDRResource) IsFull() bool {
	return len(c.Used) == c.Size
}

func (c *CIDRResource) Contains(ip net.IP) bool {
	if ip.To4() == nil {
		return false
	}

	if c.CIDR != nil {
		return c.CIDR.Contains(ip)
	}

	start := binary.BigEndian.Uint32(c.FirstIP.To4())
	end := binary.BigEndian.Uint32(c.LastIP.To4())
	value := binary.BigEndian.Uint32(ip.To4())
	return start <= value && value <= end
}

func (c *CIDRResource) IntersectsWith(value *CIDRResource) bool {
	if value == nil {
		return false
	}

	cf, cl := c.FirstIP.To4(), c.LastIP.To4()
	vf, vl := value.FirstIP.To4(), value.LastIP.To4()

	if cf == nil || cl == nil || vf == nil || vl == nil {
		return false
	}

	cfn := binary.BigEndian.Uint32(cf)
	cln := binary.BigEndian.Uint32(cl)
	vfn := binary.BigEndian.Uint32(vf)
	vln := binary.BigEndian.Uint32(vl)

	if cln < vfn || vln < cfn {
		return false
	}

	return true
}

func (c *CIDRResource) NewIPRange() IPRange {
	// note: currently we only support IPv4
	first := c.FirstIP.To4()
	last := c.LastIP.To4()

	if first == nil || last == nil {
		return nil
	}

	return &IPv4Range{
		first: binary.BigEndian.Uint32(first),
		last:  binary.BigEndian.Uint32(last),
	}
}

type IPRange interface {
	First() net.IP
	Last() net.IP
	Curr() net.IP
	Next() net.IP
	Prev() net.IP
}

type IPv4Range struct {
	first uint32
	last  uint32
	index uint32
}

func (r *IPv4Range) First() net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, r.first)
	return ip
}

func (r *IPv4Range) Last() net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, r.last)
	return ip
}

func (r *IPv4Range) Curr() net.IP {
	curr := r.first + r.index
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, curr)
	return ip
}

func (r *IPv4Range) Next() net.IP {
	if r.index == math.MaxUint32 ||
		int64(r.first)+int64(r.index) == int64(r.last) {
		return nil
	}
	r.index++
	return r.Curr()
}

func (r *IPv4Range) Prev() net.IP {
	if r.index == 0 {
		return nil
	}
	r.index--
	return r.Curr()
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

	first, last, ipnet, err := util.GetAddressRange(eip)

	if err != nil {
		return err
	}

	if _, ok := d.IPPool[name]; ok {
		d.Log.Info("Cannot add eips with same name")
		return errors.DataStoreEIPDuplicateError{CIDR: eip}
	}
	if d.IsInteractOfCurrentPool(ipnet, first, last) {
		return errors.DataStoreEIPDuplicateError{CIDR: eip}
	}
	d.lock.Lock()
	defer d.lock.Unlock()

	d.IPPool[name] = &CIDRResource{
		EIPRefName:    name,
		CIDR:          ipnet,
		FirstIP:       first,
		LastIP:        last,
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
		rg := ips.NewIPRange()
		if rg == nil {
			continue
		}

		for curr := rg.First(); curr != nil; curr = rg.Next() {
			if !ips.UsingKnownIPs {
				last := curr.To4()[3]
				if last == 0 || last == 255 {
					continue
				}
			}
			if _, ok := ips.Used[curr.String()]; !ok {
				d.lock.Lock()
				defer d.lock.Unlock()
				ips.Used[curr.String()] = &EIPRef{
					EIPRefName: ips.EIPRefName,
					Address:    curr.String(),
					Service:    types.NamespacedName{Name: serviceName, Namespace: ns},
				}
				selectIP.Address = curr.String()
				selectIP.EIPRefName = ips.EIPRefName
				d.Log.Info("Assign IP to service", "Service", serviceName, "ip", selectIP.Address)
				return selectIP, nil
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
