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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	EipDeleteReason      = "delete eip"
	EipAddOrUpdateReason = "add/update eip"
)

type Record struct {
	// The IP address specified by the service
	IP string
	// The Eip name specified by the service
	Eip string
	// The Protocol specified by the service, should be deprecated
	Protocol string
}

type Result struct {
	// The result of ip allocation
	Record *Record
}

type Request struct {
	// Service.Namespace + Service.Name
	Key string

	// The Allocate records specifying allocation
	Allocate *Record

	// The Release records specifying release
	Release *Record
}

func (i *Request) assignIPFromEip(eip *networkv1alpha2.Eip) string {
	if !eip.DeletionTimestamp.IsZero() {
		return ""
	}

	if i.Allocate == nil {
		return ""
	}

	if i.Allocate.Protocol != eip.GetProtocol() {
		return ""
	}

	if i.Allocate.Eip != eip.Name && i.Allocate.IP != "" {
		return ""
	}

	if eip.Spec.Disable || (!eip.Status.Ready && eip.GetProtocol() == constant.OpenELBProtocolLayer2) {
		return ""
	}

	for addr, svcs := range eip.Status.Used {
		tmp := strings.Split(svcs, ";")
		for _, svc := range tmp {
			if svc == i.Key {
				return addr
			}
		}
	}

	ip := net.ParseIP(i.Allocate.IP)
	offset := 0
	if ip != nil {
		offset = eip.IPToOrdinal(ip)
		if offset < 0 {
			return ""
		}
	}

	for ; offset < eip.Status.PoolSize; offset++ {
		addr := cnet.IncrementIP(*cnet.ParseIP(eip.Status.FirstIP), big.NewInt(int64(offset))).String()
		tmp, ok := eip.Status.Used[addr]
		if !ok {
			if eip.Status.Used == nil {
				eip.Status.Used = make(map[string]string)
			}
			eip.Status.Used[addr] = i.Key
			eip.Status.Usage = len(eip.Status.Used)
			if eip.Status.Usage == eip.Status.PoolSize {
				eip.Status.Occupied = true
			}
			return addr
		} else {
			if ip != nil {
				eip.Status.Used[addr] = fmt.Sprintf("%s;%s", tmp, i.Key)
				return addr
			}
		}
	}

	return ""
}

// look up by key in IPAMRequest
func (i *Request) releaseIPFromEip(eip *networkv1alpha2.Eip) {
	if !eip.DeletionTimestamp.IsZero() {
		return
	}

	for addr, svcs := range eip.Status.Used {
		tmp := strings.Split(svcs, ";")
		for _, svc := range tmp {
			if svc != i.Key {
				continue
			}

			if len(tmp) == 1 {
				delete(eip.Status.Used, addr)
				eip.Status.Usage = len(eip.Status.Used)
				if eip.Status.Usage != eip.Status.PoolSize {
					eip.Status.Occupied = false
				}
			} else {
				eip.Status.Used[addr] = strings.Join(util.RemoveString(tmp, i.Key), ";")
			}

		}
	}
}

func (i *Request) getAllocatedEIPInfo() (string, string, error) {
	eips := &networkv1alpha2.EipList{}
	err := Allocator.List(context.Background(), eips)
	if err != nil {
		return "", "", err
	}

	for _, eip := range eips.Items {
		if !eip.DeletionTimestamp.IsZero() {
			return "", "", nil
		}

		for addr, used := range eip.Status.Used {
			svcs := strings.Split(used, ";")
			for _, svc := range svcs {
				if svc == i.Key {
					return eip.Name, addr, nil
				}
			}
		}
	}

	return "", "", nil
}

func (i *Request) ConstructAllocate(svc *v1.Service) error {
	if svc == nil || svc.Annotations == nil {
		return nil
	}

	if value, ok := svc.Annotations[constant.OpenELBAnnotationKey]; !ok || value != constant.OpenELBAnnotationValue {
		return nil
	}

	if !svc.DeletionTimestamp.IsZero() || svc.Spec.Type != v1.ServiceTypeLoadBalancer {
		return nil
	}

	eipName, addr, err := i.getAllocatedEIPInfo()
	if err != nil {
		return err
	}

	svcEIP := svc.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2]
	svcLBIP := svc.Spec.LoadBalancerIP
	if value, ok := svc.Annotations[constant.OpenELBEIPAnnotationKey]; ok {
		svcLBIP = value
	}
	if svcEIP == eipName && svcLBIP == addr {
		return nil
	}

	i.Allocate = &Record{}
	i.Allocate.Eip = svcEIP
	i.Allocate.IP = svcLBIP
	i.Allocate.Protocol = constant.OpenELBProtocolBGP
	if protocol, ok := svc.Annotations[constant.OpenELBProtocolAnnotationKey]; ok {
		i.Allocate.Protocol = protocol
	}

	return nil
}

func (i *Request) ConstructRelease(svc *v1.Service) error {
	if svc == nil || svc.Annotations == nil {
		return nil
	}

	if value, ok := svc.Annotations[constant.OpenELBAnnotationKey]; !ok || value != constant.OpenELBAnnotationValue {
		return nil
	}

	if svc.Spec.Type != v1.ServiceTypeLoadBalancer {
		return nil
	}

	eipName, addr, err := i.getAllocatedEIPInfo()
	if err != nil {
		return err
	}

	svcLBIP := ""
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		svcLBIP = svc.Status.LoadBalancer.Ingress[0].IP
	}

	svcEIP := svc.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2]
	if svcEIP == eipName && svcLBIP == addr && svc.DeletionTimestamp.IsZero() {
		return nil
	}

	// new allocation
	if svcLBIP != addr && svcLBIP == "" && svc.DeletionTimestamp.IsZero() {
		return nil
	}

	if eipName == "" && addr == "" {
		return nil
	}

	i.Release = &Record{Eip: eipName, IP: addr}
	return nil
}

var (
	Allocator *IPAM
)

type IPAM struct {
	client.Client
	log logr.Logger
	record.EventRecorder
}

const name = "IPAM"

func SetupIPAM(mgr ctrl.Manager) error {
	Allocator = &IPAM{
		log:           ctrl.Log.WithName(name),
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor(name),
	}

	return ctrl.NewControllerManagedBy(mgr).Named(name).
		For(&networkv1alpha2.Eip{}).WithEventFilter(predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldEip := e.ObjectOld.(*networkv1alpha2.Eip)
			newEip := e.ObjectNew.(*networkv1alpha2.Eip)

			if !reflect.DeepEqual(oldEip.DeletionTimestamp, newEip.DeletionTimestamp) {
				return true
			}

			if !reflect.DeepEqual(oldEip.Spec, newEip.Spec) {
				return true
			}

			return false
		},
	}).Complete(Allocator)
}

// +kubebuilder:rbac:groups=network.kubesphere.io,resources=eips,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.kubesphere.io,resources=eips/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

func (i *IPAM) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	i.log.Info("start setup openelb eip")
	defer i.log.Info("finish reconcile openelb eip")

	eip := &networkv1alpha2.Eip{}

	err := i.Get(ctx, req.NamespacedName, eip)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if util.IsDeletionCandidate(eip, constant.IPAMFinalizerName) {
		if err := i.removeEip(eip); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(eip, constant.IPAMFinalizerName)
		return ctrl.Result{}, i.Update(context.Background(), eip)
	}

	if util.NeedToAddFinalizer(eip, constant.IPAMFinalizerName) {
		controllerutil.AddFinalizer(eip, constant.IPAMFinalizerName)
		if err := i.Update(ctx, eip); err != nil {
			return ctrl.Result{}, err
		}
	}

	clone := eip.DeepCopy()
	if err = i.updateEip(clone); err != nil {
		i.Event(eip, v1.EventTypeWarning, EipAddOrUpdateReason, fmt.Sprintf("%s: %s", util.GetNodeName(), err.Error()))
		return ctrl.Result{}, err
	}

	if reflect.DeepEqual(clone.Status, eip.Status) {
		return ctrl.Result{}, nil
	}
	//i.updateMetrics(eip)
	return ctrl.Result{}, i.Client.Status().Update(context.Background(), clone)
}

func (i *IPAM) AssignIP(request *Request) (result Result, err error) {
	eip := &networkv1alpha2.Eip{}
	err = i.Get(context.Background(), types.NamespacedName{Name: request.Allocate.Eip}, eip)
	if err != nil {
		return result, err
	}

	clone := eip.DeepCopy()
	if !eip.Status.Ready && eip.GetProtocol() == constant.OpenELBProtocolLayer2 {
		return result, fmt.Errorf("layer2 eip:%s speaker not ready", eip.Name)
	}

	addr := request.assignIPFromEip(clone)
	if addr == "" {
		return result, fmt.Errorf("no avliable eip")
	}
	// i.updateMetrics(clone)
	if !reflect.DeepEqual(clone, eip) {
		if err := i.Client.Status().Update(context.Background(), clone); err != nil {
			return result, err
		}
	}

	result.Record = request.Allocate
	if result.Record.IP == "" {
		result.Record.IP = addr
	}
	i.log.Info("assign ip", "request", request, "eip status", clone.Status)

	return result, nil
}

func (i *IPAM) updateEip(e *networkv1alpha2.Eip) error {
	if e.Status.FirstIP == "" {
		base, size, _ := e.GetSize()
		e.Status.PoolSize = int(size)
		e.Status.FirstIP = base.String()
		e.Status.LastIP = cnet.IncrementIP(cnet.IP{IP: base}, big.NewInt(size-1)).String()
		if base.To4() != nil {
			e.Status.V4 = true
		}
	}

	return i.syncEip(e)
}

func (i *IPAM) syncEip(e *networkv1alpha2.Eip) error {
	tmp := make(map[string]string)
	for k, v := range e.Status.Used {
		svcs := strings.Split(v, ";")
		tmpV := ""
		for _, svc := range svcs {
			strs := strings.Split(svc, "/")
			obj := v1.Service{}
			err := i.Get(context.Background(), client.ObjectKey{Namespace: strs[0], Name: strs[1]}, &obj)
			if err != nil && !k8serrors.IsNotFound(err) {
				return err
			}

			if err == nil {
				if tmpV == "" {
					tmpV = svc
				} else {
					tmpV = tmpV + ";" + svc
				}
			}
		}

		if tmpV != "" {
			tmp[k] = tmpV
		}
	}
	e.Status.Used = tmp
	e.Status.Usage = len(tmp)
	if e.Status.Usage < e.Status.PoolSize {
		e.Status.Occupied = false
	} else {
		e.Status.Occupied = true
	}

	return nil
}

func (i *IPAM) removeEip(e *networkv1alpha2.Eip) error {
	svcs := v1.ServiceList{}
	opts := labels.SelectorFromSet(labels.Set(map[string]string{constant.OpenELBEIPAnnotationKeyV1Alpha2: e.Name}))
	err := i.List(context.Background(), &svcs, &client.ListOptions{LabelSelector: opts})
	if err != nil {
		return err
	}

	for _, svc := range svcs.Items {
		delete(svc.Labels, constant.OpenELBEIPAnnotationKeyV1Alpha2)
		if err := i.Update(context.Background(), &svc); err != nil {
			return err
		}
	}

	return nil
}

func (i *IPAM) ReleaseIP(request *Request) error {
	eip := &networkv1alpha2.Eip{}
	err := i.Get(context.Background(), types.NamespacedName{Name: request.Release.Eip}, eip)
	if err != nil {
		return err
	}

	clone := eip.DeepCopy()
	request.releaseIPFromEip(clone)
	//i.updateMetrics(clone)
	if !reflect.DeepEqual(clone, eip) {
		if err := i.Client.Status().Update(context.Background(), clone); err != nil {
			return err
		}
	}

	i.log.Info("release ip", "request", request, "eip status", clone.Status)
	return nil
}

func (i *IPAM) updateMetrics(eip *networkv1alpha2.Eip) {
	total := float64(eip.Status.PoolSize)
	used := float64(eip.Status.Usage)
	var svcCount float64 = 0
	for _, svc := range eip.Status.Used {
		svcCount = svcCount + float64(len(strings.Split(svc, ";")))
	}

	metrics.UpdateEipMetrics(eip.Name, total, used, svcCount)
}
