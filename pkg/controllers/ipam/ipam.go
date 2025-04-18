package ipam

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"sort"
	"strings"

	"github.com/openelb/openelb/pkg/util/iprange"

	networkv1alpha2 "github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/metrics"
	"github.com/openelb/openelb/pkg/util"
	cnet "github.com/openelb/openelb/pkg/util/net"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EipDeleteReason      = "delete eip"
	EipAddOrUpdateReason = "add/update eip"
	EipNotContainIP      = "no available eip was found containing the ip"
)

type Manager struct {
	client.Client
	record.EventRecorder
}

type svcRecord struct {
	// Service.Namespace + Service.Name
	Key string
	// The IP address specified by the service
	IP string
	// The Eip name specified by the service
	Eip string
}

func (s svcRecord) String() string {
	return fmt.Sprintf("service:%s, ip:%s, eip:%s", s.Key, s.IP, s.Eip)
}

// The result of ip allocation
type Result *svcRecord

type Request struct {
	// The Allocate records specifying allocation
	Allocate *svcRecord

	// The Release records specifying release
	Release *svcRecord
}

func NewManager(client client.Client) *Manager {
	return &Manager{
		Client: client,
	}
}

func (i *Manager) assignIPFromEip(allocate *svcRecord, eip *networkv1alpha2.Eip) (string, error) {
	if allocate == nil {
		return "", fmt.Errorf("allocate is nil")
	}

	if !eip.DeletionTimestamp.IsZero() {
		return "", fmt.Errorf("eip:%s is deleting", eip.Name)
	}

	if eip.Spec.Disable {
		return "", fmt.Errorf("eip:%s is disabled", eip.Name)
	}

	for addr, svcs := range eip.Status.Used {
		tmp := strings.Split(svcs, ";")
		for _, svc := range tmp {
			if svc == allocate.Key && allocate.IP == addr {
				return addr, nil
			}
		}
	}

	ip := net.ParseIP(allocate.IP)
	offset := 0
	if ip != nil {
		offset = eip.IPToOrdinal(ip)
		if offset < 0 {
			return "", fmt.Errorf("the specified ip:%s is beyond the range of eip[%s:%s]", allocate.IP, eip.Name, eip.Spec.Address)
		}
	}

	for ; offset < eip.Status.PoolSize; offset++ {
		addr := cnet.IncrementIP(*cnet.ParseIP(eip.Status.FirstIP), big.NewInt(int64(offset))).String()
		tmp, ok := eip.Status.Used[addr]
		if !ok {
			if eip.Status.Used == nil {
				eip.Status.Used = make(map[string]string)
			}
			eip.Status.Used[addr] = allocate.Key
			eip.Status.Usage = len(eip.Status.Used)
			if eip.Status.Usage == eip.Status.PoolSize {
				eip.Status.Occupied = true
			}
			return addr, nil
		} else {
			if ip != nil {
				eip.Status.Used[addr] = fmt.Sprintf("%s;%s", tmp, allocate.Key)
				return addr, nil
			}
		}
	}

	return "", fmt.Errorf("no suitable ip to allocate")
}

// look up by key in IPAMRequest
func (i *Manager) releaseIPFromEip(svcInfo string, eip *networkv1alpha2.Eip) {
	if !eip.DeletionTimestamp.IsZero() {
		return
	}

	for addr, svcs := range eip.Status.Used {
		tmp := strings.Split(svcs, ";")
		for _, svc := range tmp {
			if svc != svcInfo {
				continue
			}

			if len(tmp) == 1 {
				delete(eip.Status.Used, addr)
				eip.Status.Usage = len(eip.Status.Used)
				if eip.Status.Usage != eip.Status.PoolSize {
					eip.Status.Occupied = false
				}
			} else {
				eip.Status.Used[addr] = strings.Join(util.RemoveString(tmp, svcInfo), ";")
			}

		}
	}
}

func (i *Manager) getAllocatedEIPInfo(ctx context.Context, svcInfo string) (string, string, error) {
	eips := &networkv1alpha2.EipList{}
	err := i.List(ctx, eips)
	if err != nil {
		return "", "", err
	}

	for _, eip := range eips.Items {
		if !eip.DeletionTimestamp.IsZero() {
			continue
		}

		for addr, used := range eip.Status.Used {
			svcs := strings.Split(used, ";")
			for _, svc := range svcs {
				if svc == svcInfo {
					return eip.Name, addr, nil
				}
			}
		}
	}

	return "", "", nil
}

type info struct {
	svcName        string
	svcSpecifyEIP  string
	svcSpecifyLBIP string
	svcStatusLBIP  string
	allocatedEip   string
	allocatedIP    string
}

func (i *info) needUpdate() bool {
	if i.svcSpecifyEIP == "" && i.svcSpecifyLBIP == "" && i.svcStatusLBIP == i.allocatedIP {
		return false
	}

	if i.svcSpecifyEIP == i.allocatedEip {
		if i.svcSpecifyLBIP == i.allocatedIP && i.svcStatusLBIP != "" {
			return false
		}

		if i.svcSpecifyLBIP == "" && i.svcStatusLBIP == i.allocatedIP {
			return false
		}
	}

	return true
}

func (i *Manager) ConstructRequest(ctx context.Context, svc *v1.Service) (req Request, err error) {
	if svc == nil || svc.Annotations == nil {
		return Request{}, nil
	}

	info := info{svcName: types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}.String()}
	info.allocatedEip, info.allocatedIP, err = i.getAllocatedEIPInfo(ctx, info.svcName)
	if err != nil {
		return Request{}, err
	}

	info.svcStatusLBIP = ""
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		var ips []string
		for _, i := range svc.Status.LoadBalancer.Ingress {
			if i.IP != "" {
				ips = append(ips, i.IP)
			}

			if i.Hostname != "" {
				ips = append(ips, i.Hostname)
			}
		}

		info.svcStatusLBIP = strings.Join(ips, ";")
	}

	req.Release = i.constructRelease(info)
	if needRelease(svc) {
		klog.V(4).Infof("Only need Release service loadbalanceip")
		return req, nil
	}

	info.svcSpecifyLBIP = svc.Spec.LoadBalancerIP
	if value, ok := svc.Annotations[constant.OpenELBEIPAnnotationKey]; ok {
		info.svcSpecifyLBIP = value
	}

	info.svcSpecifyEIP = svc.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2]
	if info.svcSpecifyEIP == "" {
		eip, err := i.getEIP(context.Background(), svc.Namespace, info.svcSpecifyLBIP, info.svcSpecifyEIP)
		if err != nil {
			i.Eventf(svc, v1.EventTypeWarning, "ConstructRequest", "failed to construct allocate request: %s", err.Error())
			klog.Errorf("get eip error:%s", err.Error())
			return req, nil
		}
		info.svcSpecifyEIP = eip.Name
	}

	_, exist := svc.Labels[constant.OpenELBEIPAnnotationKeyV1Alpha2]
	if !info.needUpdate() && exist {
		klog.V(4).Info("no need update service loadbalanceip")
		return Request{}, nil
	}

	req.Allocate = &svcRecord{
		Key: info.svcName,
		Eip: info.svcSpecifyEIP,
		IP:  info.svcSpecifyLBIP,
	}
	return req, nil
}

func needRelease(svc *v1.Service) bool {
	if svc == nil || svc.Annotations == nil {
		return true
	}

	if !svc.DeletionTimestamp.IsZero() {
		return true
	}

	if svc.Spec.Type != v1.ServiceTypeLoadBalancer {
		return true
	}

	if value, ok := svc.Annotations[constant.OpenELBAnnotationKey]; !ok || value != constant.OpenELBAnnotationValue {
		return true
	}

	return false
}

func (i *Manager) getEIP(ctx context.Context, ns string, svcip string, specifyEip string) (*networkv1alpha2.Eip, error) {
	if specifyEip == "" {
		if svcip != "" {
			return i.getEIPBasedOnIP(ctx, svcip)
		}
		return i.getDefaultEIP(ctx, ns)
	}

	eip := &networkv1alpha2.Eip{}
	if err := i.Get(ctx, types.NamespacedName{Name: specifyEip}, eip); err != nil {
		return nil, err
	}

	if !eip.DeletionTimestamp.IsZero() {
		return nil, fmt.Errorf("eip:%s is deleting", eip.Name)
	}

	if eip.Spec.Disable {
		return nil, fmt.Errorf("eip:%s is disabled", eip.Name)
	}

	if svcip != "" && !eip.Contains(net.ParseIP(svcip)) {
		return nil, fmt.Errorf(EipNotContainIP+":[%s]", svcip)
	}

	return eip, nil
}

func (i *Manager) getEIPBasedOnIP(ctx context.Context, ip string) (*networkv1alpha2.Eip, error) {
	eips := &networkv1alpha2.EipList{}
	if err := i.List(ctx, eips); err != nil {
		return nil, err
	}

	for _, e := range eips.Items {
		if e.Contains(net.ParseIP(ip)) {
			if !e.DeletionTimestamp.IsZero() {
				return nil, fmt.Errorf("eip:%s is deleting", e.Name)
			}

			if e.Spec.Disable {
				return nil, fmt.Errorf("eip:%s is disabled", e.Name)
			}

			return e.DeepCopy(), nil
		}
	}

	return nil, fmt.Errorf(EipNotContainIP+":[%s]", ip)
}

func (i *Manager) getDefaultEIP(ctx context.Context, name string) (*networkv1alpha2.Eip, error) {
	// get namespace info
	ns := &v1.Namespace{}
	if err := i.Get(ctx, types.NamespacedName{Name: name}, ns); err != nil {
		return nil, err
	}

	// get namespace dafault eip
	eips := &networkv1alpha2.EipList{}
	if err := i.List(ctx, eips); err != nil {
		return nil, err
	}

	var defaultEip *networkv1alpha2.Eip
	nseips := make([]*networkv1alpha2.Eip, 0)
	for _, e := range eips.Items {
		if !e.DeletionTimestamp.IsZero() || e.Spec.Disable {
			continue
		}

		for _, n := range e.Spec.Namespaces {
			if n == name {
				nseips = append(nseips, e.DeepCopy())
				break
			}
		}

		if ns.Labels != nil && e.Spec.NamespaceSelector != nil {
			s := metav1.SetAsLabelSelector(e.Spec.NamespaceSelector)
			l, err := metav1.LabelSelectorAsSelector(s)
			if err != nil {
				return nil, fmt.Errorf("eip:[%s] invalid namespace label selector %v", e.Name, s)
			}

			nsLabels := labels.Set(ns.Labels)
			if l.Matches(nsLabels) {
				nseips = append(nseips, e.DeepCopy())
			}
		}

		if defaultEip == nil && e.IsDefault() {
			defaultEip = e.DeepCopy()
		}
	}

	if len(nseips) != 0 {
		sort.Slice(nseips, func(i, j int) bool {
			if nseips[i].Status.Occupied != nseips[j].Status.Occupied {
				return !nseips[i].Status.Occupied
			}

			return nseips[i].Spec.Priority < nseips[j].Spec.Priority
		})

		allEIPStr := []string{}
		for _, e := range nseips {
			allEIPStr = append(allEIPStr, e.Name)
		}
		klog.V(1).Infof("all available eips in weight order is: %s", strings.Join(allEIPStr, ","))
		klog.V(1).Infof("auto select eip[%s] for allocation", nseips[0].Name)
		return nseips[0], nil
	}

	if defaultEip == nil {
		return defaultEip, fmt.Errorf("no available default eip found")
	}

	return defaultEip, nil
}

func (i *Manager) constructRelease(info info) *svcRecord {
	r := &svcRecord{
		Key: info.svcName,
		Eip: info.allocatedEip,
		IP:  info.allocatedIP,
	}

	if info.allocatedEip == "" && info.allocatedIP == "" {
		// no eip record and no status record
		if info.svcStatusLBIP == "" {
			return nil
		}

		// eip delete first - no eip record
		r.IP = info.svcStatusLBIP
	}

	return r
}

func (i *Manager) AssignIP(ctx context.Context, ipFamilies []v1.IPFamily, allocate *svcRecord) error {
	if allocate == nil {
		return nil
	}

	// assign ip from eip
	eip := &networkv1alpha2.Eip{}
	err := i.Get(ctx, types.NamespacedName{Name: allocate.Eip}, eip)
	if err != nil {
		return err
	}

	parseRange, err := iprange.ParseRange(eip.Spec.Address)
	if err != nil {
		return err
	}
	eipFamily := parseRange.Family()
	if !IsSameFamily(eipFamily, ipFamilies) {
		return fmt.Errorf("service can't use different family eip")
	}

	clone := eip.DeepCopy()
	addr, err := i.assignIPFromEip(allocate, clone)
	if err != nil {
		return fmt.Errorf("no avliable eip, err:%s", err.Error())
	}
	// i.updateMetrics(clone)
	if !reflect.DeepEqual(clone, eip) {
		if err := i.Client.Status().Update(ctx, clone); err != nil {
			return err
		}
	}

	allocate.IP = addr
	return nil
}

func (i *Manager) ReleaseIP(ctx context.Context, release *svcRecord) error {
	if release == nil {
		return nil
	}

	eip := &networkv1alpha2.Eip{}
	err := i.Get(ctx, types.NamespacedName{Name: release.Eip}, eip)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	clone := eip.DeepCopy()
	i.releaseIPFromEip(release.Key, clone)
	//i.updateMetrics(clone)
	if !reflect.DeepEqual(clone, eip) {
		if err := i.Status().Update(ctx, clone); err != nil {
			klog.Errorf(err.Error())
			return err
		}
	}

	return nil
}

func (i *Manager) updateMetrics(eip *networkv1alpha2.Eip) {
	total := float64(eip.Status.PoolSize)
	used := float64(eip.Status.Usage)
	var svcCount float64 = 0
	for _, svc := range eip.Status.Used {
		svcCount = svcCount + float64(len(strings.Split(svc, ";")))
	}

	metrics.UpdateEipMetrics(eip.Name, total, used, svcCount)
}

func IsSameFamily(eipFamily iprange.Family, ipFamilies []v1.IPFamily) bool {
	for _, ipFamily := range ipFamilies {
		if ipFamily == v1.IPv4Protocol && eipFamily == iprange.V4Family {
			return true
		}
		if ipFamily == v1.IPv6Protocol && eipFamily == iprange.V6Family {
			return true
		}
	}
	return false
}
