package ipam

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"sort"
	"strings"

	"github.com/go-logr/logr"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EipDeleteReason      = "delete eip"
	EipAddOrUpdateReason = "add/update eip"
	EipNotContainIP      = "no available eip was found containing the ip"
)

type Manager struct {
	client.Client
	log logr.Logger
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
		log:    ctrl.Log.WithName("IPAM"),
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
			if svc == allocate.Key {
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
		if i.svcSpecifyLBIP == i.allocatedIP {
			return false
		}

		if i.svcSpecifyLBIP == "" && i.svcStatusLBIP == i.allocatedIP {
			return false
		}
	}

	return true
}

func (i *Manager) ConstructRequest(ctx context.Context, svc *v1.Service) (Request, error) {
	req := Request{}
	if svc == nil || svc.Annotations == nil {
		return req, nil
	}

	svcSpecifyLBIP := svc.Spec.LoadBalancerIP
	if value, ok := svc.Annotations[constant.OpenELBEIPAnnotationKey]; ok {
		svcSpecifyLBIP = value
	}

	svcSpecifyEIP := svc.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2]
	if svcSpecifyEIP == "" {
		eip, err := i.getEIP(context.Background(), svc.Namespace, svcSpecifyLBIP, svcSpecifyEIP)
		if err != nil || eip == nil {
			i.log.Error(err, "get eip error")
			return req, err
		}
		svcSpecifyEIP = eip.Name
	}

	eip, addr, err := i.getAllocatedEIPInfo(ctx, types.NamespacedName{
		Namespace: svc.Namespace, Name: svc.Name}.String())
	if err != nil {
		return req, nil
	}

	lbIP := ""
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		var ips []string
		for _, i := range svc.Status.LoadBalancer.Ingress {
			ips = append(ips, i.IP)
		}

		lbIP = strings.Join(ips, ";")
	}

	info := info{
		svcName:        types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}.String(),
		svcSpecifyEIP:  svcSpecifyEIP,
		svcSpecifyLBIP: svcSpecifyLBIP,
		allocatedIP:    addr,
		allocatedEip:   eip,
		svcStatusLBIP:  lbIP,
	}

	if needRelease(svc) {
		req.Release = i.constructRelease(info)
		return req, nil
	}

	if !info.needUpdate() {
		if _, exist := svc.Labels[constant.OpenELBEIPAnnotationKeyV1Alpha2]; !exist {
			req.Allocate = &svcRecord{
				Key: info.svcName,
				Eip: info.allocatedEip,
				IP:  info.allocatedIP,
			}
		}
		return req, nil
	}

	req.Release = i.constructRelease(info)
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
			if nseips[i].Spec.Disable != nseips[j].Spec.Disable {
				return !nseips[i].Spec.Disable
			}

			if nseips[i].Status.Occupied != nseips[j].Status.Occupied {
				return !nseips[i].Status.Occupied
			}

			return nseips[i].Spec.Priority < nseips[j].Spec.Priority
		})

		i.log.V(1).Info("auto select eip for allocation", "eip", nseips[0].Name, "all eip", nseips)
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

func (i *Manager) AssignIP(ctx context.Context, allocate *svcRecord) error {
	if allocate == nil {
		return nil
	}

	// assign ip from eip
	eip := &networkv1alpha2.Eip{}
	err := i.Get(ctx, types.NamespacedName{Name: allocate.Eip}, eip)
	if err != nil {
		return err
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
	i.log.Info("assign ip", "allocate Record", allocate, "eip status", clone.Status)
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
		if err := i.Client.Status().Update(ctx, clone); err != nil {
			return err
		}
	}

	i.log.Info("release ip", "release Record", release, "eip status", clone.Status)
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
