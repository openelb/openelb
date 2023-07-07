package ipam

import (
	"context"
	"fmt"
	"math/big"
	"net"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	networkv1alpha2 "github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/metrics"
	"github.com/openelb/openelb/pkg/util"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	EipDeleteReason      = "delete eip"
	EipAddOrUpdateReason = "add/update eip"
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
	// The Protocol specified by the service, should be deprecated
	Protocol string
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

	if allocate.Protocol != eip.GetProtocol() {
		return "", fmt.Errorf("eip's protocol:[%s] is not match wanted[%s]", eip.GetProtocol(), allocate.Protocol)
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

func (i *Manager) getAllocatedEIPInfo(ctx context.Context, svcInfo string) (*networkv1alpha2.Eip, string, error) {
	eips := &networkv1alpha2.EipList{}
	err := i.List(ctx, eips)
	if err != nil {
		return nil, "", err
	}

	for _, eip := range eips.Items {
		for addr, used := range eip.Status.Used {
			svcs := strings.Split(used, ";")
			for _, svc := range svcs {
				if svc == svcInfo {
					return eip.DeepCopy(), addr, nil
				}
			}
		}
	}

	return nil, "", nil
}

type info struct {
	key           string
	svcEIP        string
	svcLBIP       string
	svcStatusLBIP string
	actualEip     string
	actualIP      string
	protocol      string
	loadbalance   bool
	deleting      bool
}

func (i *Manager) ConstructRequest(ctx context.Context, svc *v1.Service) (*Request, error) {
	if svc == nil || svc.Annotations == nil {
		return nil, nil
	}

	svcEIPName, ok := svc.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2]
	if !ok || svcEIPName == "" {
		return nil, nil
	}
	svcLBIP := svc.Spec.LoadBalancerIP
	if value, ok := svc.Annotations[constant.OpenELBEIPAnnotationKey]; ok {
		svcLBIP = value
	}

	protocol := constant.OpenELBProtocolBGP
	if p, ok := svc.Annotations[constant.OpenELBProtocolAnnotationKey]; ok {
		protocol = p
	}

	eip, addr, err := i.getAllocatedEIPInfo(ctx, types.NamespacedName{
		Namespace: svc.Namespace, Name: svc.Name}.String())
	if err != nil {
		return nil, err
	}

	svcEip := &networkv1alpha2.Eip{}
	if err := i.Get(ctx, types.NamespacedName{Name: svcEIPName}, svcEip); err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	info := info{
		key:         types.NamespacedName{Namespace: svc.Namespace, Name: svc.Name}.String(),
		svcEIP:      svcEIPName,
		svcLBIP:     svcLBIP,
		actualIP:    addr,
		protocol:    protocol,
		loadbalance: svc.Spec.Type == v1.ServiceTypeLoadBalancer,
		deleting:    !svc.DeletionTimestamp.IsZero(),
	}

	if eip != nil {
		info.actualEip = eip.Name
	}

	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		var ips []string
		for _, i := range svc.Status.LoadBalancer.Ingress {
			ips = append(ips, i.IP)
		}

		info.svcStatusLBIP = strings.Join(ips, ";")
	}

	// may update eip
	if svcLBIP == "" {
		info.svcLBIP = getServiceLBIP(svcEip, svc.Status.LoadBalancer.Ingress)
	}

	if value, ok := svc.Annotations[constant.OpenELBAnnotationKey]; !ok || value != constant.OpenELBAnnotationValue {
		return &Request{
			Release: i.constructRelease(info, false),
		}, nil
	}

	return &Request{
		Allocate: i.constructAllocate(info),
		Release:  i.constructRelease(info, true),
	}, nil
}

func getServiceLBIP(eip *networkv1alpha2.Eip, status []v1.LoadBalancerIngress) string {
	if eip == nil {
		return ""
	}

	var ips []string
	if len(status) > 0 {
		for _, i := range status {
			if eip.Contains(net.ParseIP(i.IP)) {
				ips = append(ips, i.IP)
			}
		}
	}

	return strings.Join(ips, ";")
}

func (i *Manager) constructAllocate(info info) *svcRecord {
	if info.deleting || !info.loadbalance {
		return nil
	}

	if info.svcEIP == info.actualEip {
		if info.svcEIP == "" {
			return nil
		}

		if info.svcLBIP == info.actualIP {
			return nil
		}
	}

	return &svcRecord{
		Key:      info.key,
		Eip:      info.svcEIP,
		IP:       info.svcLBIP,
		Protocol: info.protocol,
	}
}

func (i *Manager) constructRelease(info info, specifyOpenELB bool) *svcRecord {
	if info.svcEIP == info.actualEip && !info.deleting && info.loadbalance && specifyOpenELB {
		// no change
		if info.svcLBIP == info.actualIP {
			return nil
		}

		// delete spec.loadbalanceIP
		if info.svcLBIP != info.actualIP && info.svcLBIP == "" {
			return nil
		}
	}

	r := &svcRecord{
		Key: info.key,
		Eip: info.actualEip,
		IP:  info.actualIP,
	}

	if info.actualEip == "" && info.actualIP == "" {
		// no eip record and no status record
		if info.svcStatusLBIP == "" {
			return nil
		}

		// eip delete first - no eip record
		r.IP = info.svcStatusLBIP
	}

	return r
}

func (i *Manager) AssignIP(ctx context.Context, allocate *svcRecord, svc *v1.Service) error {
	if allocate == nil || svc == nil {
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

	// assign ip handle service
	if !util.ContainsString(svc.Finalizers, constant.FinalizerName) {
		controllerutil.AddFinalizer(svc, constant.FinalizerName)
	}
	//update labels
	if svc.Labels == nil {
		svc.Labels = make(map[string]string)
	}
	svc.Labels[constant.OpenELBEIPAnnotationKeyV1Alpha2] = allocate.Eip

	//update ingress status
	svc.Status.LoadBalancer.Ingress = make([]v1.LoadBalancerIngress, 0)
	svc.Status.LoadBalancer.Ingress = append(svc.Status.LoadBalancer.Ingress, v1.LoadBalancerIngress{IP: addr})

	i.log.Info("assign ip", "allocate Record", allocate, "eip status", clone.Status)

	return nil
}

func (i *Manager) ReleaseIP(ctx context.Context, release *svcRecord, svc *v1.Service) error {
	if release == nil || svc == nil {
		return nil
	}

	eip := &networkv1alpha2.Eip{}
	err := i.Get(ctx, types.NamespacedName{Name: release.Eip}, eip)
	if err != nil {
		if errors.IsNotFound(err) {
			svc.Status.LoadBalancer.Ingress = nil
			controllerutil.RemoveFinalizer(svc, constant.FinalizerName)
			delete(svc.Labels, constant.OpenELBEIPAnnotationKeyV1Alpha2)
			return nil
		}
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

	// we think only openelb handles this status
	svc.Status.LoadBalancer.Ingress = nil
	controllerutil.RemoveFinalizer(svc, constant.FinalizerName)
	delete(svc.Labels, constant.OpenELBEIPAnnotationKeyV1Alpha2)

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
