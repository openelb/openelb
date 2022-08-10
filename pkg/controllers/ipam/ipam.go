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
	"github.com/openelb/openelb/pkg/speaker"
	"github.com/openelb/openelb/pkg/speaker/layer2"
	"github.com/openelb/openelb/pkg/util"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

type IPAMArgs struct {
	// Service.Namespace + Service.Name
	// Required
	Key string
	// The IP address specified by the service
	Addr string
	// The Eip name specified by the service
	// Not available right now.
	Eip string
	// The Protocol specified by the service
	// Required
	Protocol string
	Unalloc  bool
}

// Compare the parameters and results to determine if the IP address should be retrieved.
func (i *IPAMArgs) ShouldUnAssignIP(result IPAMResult) bool {
	if result.Addr == "" {
		return false
	}

	if i.Unalloc {
		return true
	}

	if i.Protocol != result.Protocol {
		return true
	}

	if i.Addr != "" && i.Addr != result.Addr {
		return true
	}

	if i.Eip != "" && i.Eip != result.Eip {
		return true
	}

	return false
}

type IPAMResult struct {
	Addr     string
	Eip      string
	Protocol string
	Sp       speaker.Speaker
}

// Called when the service is updated or created.
func (i *IPAMResult) Assigned() bool {
	if i.Addr != "" {
		return true
	}

	return false
}

func (i *IPAMResult) Clean() {
	i.Addr = ""
}

var (
	IPAMAllocator *IPAM
)

type IPAM struct {
	client.Client
	log logr.Logger
	record.EventRecorder
}

const name = "IPAM"

func SetupIPAM(mgr ctrl.Manager) error {
	IPAMAllocator = &IPAM{
		log:           ctrl.Log.WithName(name),
		Client:        mgr.GetClient(),
		EventRecorder: mgr.GetEventRecorderFor(name),
	}

	return ctrl.NewControllerManagedBy(mgr).Named(name).
		For(&networkv1alpha2.Eip{}).WithEventFilter(predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			if util.DutyOfCNI(nil, e.Meta) {
				return false
			}
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if util.DutyOfCNI(e.MetaOld, e.MetaNew) {
				return false
			}

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
	}).Complete(IPAMAllocator)
}

// +kubebuilder:rbac:groups=network.kubesphere.io,resources=eips,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.kubesphere.io,resources=eips/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

func (i *IPAM) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	eip := &networkv1alpha2.Eip{}

	err := i.Get(context.TODO(), req.NamespacedName, eip)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if util.IsDeletionCandidate(eip, constant.IPAMFinalizerName) {
		err = i.removeEip(eip)
		if err != nil {
			return ctrl.Result{}, err
		}

		metrics.DeleteEipMetrics(eip.Name)
		controllerutil.RemoveFinalizer(eip, constant.IPAMFinalizerName)
		return ctrl.Result{}, i.Update(context.Background(), eip)
	}

	if util.NeedToAddFinalizer(eip, constant.IPAMFinalizerName) {
		controllerutil.AddFinalizer(eip, constant.IPAMFinalizerName)
		err := i.Update(context.Background(), eip)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	clone := eip.DeepCopy()

	if err = i.updateEip(clone); err != nil {
		i.Client.Status().Update(context.Background(), clone)
		i.Event(eip, v1.EventTypeWarning, EipAddOrUpdateReason, fmt.Sprintf("%s: %s", util.GetNodeName(), err.Error()))
		return ctrl.Result{}, err
	}

	if reflect.DeepEqual(clone.Status, eip.Status) {
		return ctrl.Result{}, nil
	}
	i.updateMetrics(eip)
	return ctrl.Result{}, i.Client.Status().Update(context.Background(), clone)
}

func (i *IPAM) updateEip(e *networkv1alpha2.Eip) error {
	var (
		sp  speaker.Speaker
		err error
	)

	if e.Status.FirstIP == "" {
		base, size, _ := e.GetSize()
		e.Status.PoolSize = int(size)
		e.Status.FirstIP = base.String()
		e.Status.LastIP = cnet.IncrementIP(cnet.IP{IP: base}, big.NewInt(size-1)).String()
		if base.To4() != nil {
			e.Status.V4 = true
		}
	}

	err = i.syncEip(e)
	if err != nil {
		return err
	}

	sp = speaker.GetSpeaker(e.GetSpeakerName())
	if sp == nil {
		sp, err = layer2.NewSpeaker(e.Spec.Interface, e.Status.V4)
		if err == nil {
			err = speaker.RegisterSpeaker(e.GetSpeakerName(), sp)
		}
	}
	if err != nil {
		e.Status.Ready = false
	} else {
		e.Status.Ready = true
	}

	return err
}

func (i *IPAM) syncEip(e *networkv1alpha2.Eip) error {
	var err error

	tmp := make(map[string]string)
	for k, v := range e.Status.Used {
		svcs := strings.Split(v, ";")
		tmpV := ""
		for _, svc := range svcs {
			strs := strings.Split(svc, "/")
			obj := v1.Service{}
			err = i.Get(context.Background(), client.ObjectKey{Namespace: strs[0], Name: strs[1]}, &obj)
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
	if e.Status.Usage != 0 {
		for ip, _ := range e.Status.Used {
			s := speaker.GetSpeaker(e.GetSpeakerName())
			if s == nil {
				i.log.Info("remove eip, but there is no speaker")
				break
			}

			err := s.DelBalancer(ip)
			if err != nil {
				i.log.Error(err, fmt.Sprintf("delete balancer [%s:%s] error", e.GetSpeakerName(), ip))
			}
		}
	}

	if e.Spec.Protocol == constant.OpenELBProtocolLayer2 {
		speaker.UnRegisterSpeaker(e.Spec.Interface)
	}

	svcs := v1.ServiceList{}
	opts := &client.ListOptions{}
	client.MatchingLabels{
		constant.OpenELBEIPAnnotationKeyV1Alpha2: e.Name,
	}.ApplyToList(opts)
	err := i.List(context.Background(), &svcs, opts)
	if err != nil {
		return err
	}

	for _, svc := range svcs.Items {
		clone := svc.DeepCopy()
		if clone.Labels != nil {
			delete(clone.Labels, constant.OpenELBEIPAnnotationKeyV1Alpha2)
		}
		if !reflect.DeepEqual(clone, &svc) {
			err = i.Update(context.Background(), clone)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (i *IPAM) AssignIP(args IPAMArgs) (IPAMResult, error) {
	eips := &networkv1alpha2.EipList{}
	err := i.List(context.Background(), eips)
	if err != nil {
		return IPAMResult{}, err
	}

	err = fmt.Errorf("no avliable eip")
	var result IPAMResult

	for _, eip := range eips.Items {
		clone := eip.DeepCopy()
		addr := args.assignIPFromEip(clone)
		i.updateMetrics(clone)
		if addr != "" {
			if !reflect.DeepEqual(clone, eip) {
				err = i.Client.Status().Update(context.Background(), clone)
				ctrl.Log.Info("assignIP update eip", "eip", clone.Status)
			}

			result.Addr = addr
			result.Eip = eip.Name
			result.Protocol = eip.GetProtocol()
			result.Sp = speaker.GetSpeaker(eip.GetSpeakerName())

			if result.Sp == nil {
				err = fmt.Errorf("layer2 eip speaker not ready")
			}

			break
		}
	}

	i.log.Info("assignIP",
		"args", args,
		"result", result,
		"err", err)

	return result, err
}

func (a IPAMArgs) assignIPFromEip(eip *networkv1alpha2.Eip) string {
	if eip.DeletionTimestamp != nil {
		return ""
	}

	for addr, svcs := range eip.Status.Used {
		tmp := strings.Split(svcs, ";")
		for _, svc := range tmp {
			if svc == a.Key {
				return addr
			}
		}
	}

	if a.Protocol != eip.GetProtocol() {
		return ""
	}

	if eip.Name != a.Eip && a.Eip != "" {
		return ""
	}

	if eip.Spec.Disable || !eip.Status.Ready {
		return ""
	}

	ip := net.ParseIP(a.Addr)
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
			eip.Status.Used[addr] = a.Key
			eip.Status.Usage = len(eip.Status.Used)
			if eip.Status.Usage == eip.Status.PoolSize {
				eip.Status.Occupied = true
			}
			return addr
		} else {
			if ip != nil {
				eip.Status.Used[addr] = fmt.Sprintf("%s;%s", tmp, a.Key)
				return addr
			}
		}
	}

	return ""
}

// look up by key in IPAMArgs
func (a IPAMArgs) unAssignIPFromEip(eip *networkv1alpha2.Eip, peek bool) string {
	if eip.DeletionTimestamp != nil {
		return ""
	}

	for addr, svcs := range eip.Status.Used {
		tmp := strings.Split(svcs, ";")
		for _, svc := range tmp {
			if svc == a.Key {
				if !peek {
					if len(tmp) == 1 {
						delete(eip.Status.Used, addr)
						eip.Status.Usage = len(eip.Status.Used)
						if eip.Status.Usage != eip.Status.PoolSize {
							eip.Status.Occupied = false
						}
					} else {
						eip.Status.Used[addr] = strings.Join(util.RemoveString(tmp, a.Key), ";")
					}
				}

				return addr
			}
		}
	}

	return ""
}

func (i *IPAM) UnAssignIP(args IPAMArgs, peek bool) (IPAMResult, error) {
	var result IPAMResult

	eips := &networkv1alpha2.EipList{}
	err := i.List(context.Background(), eips)
	if err != nil {
		return result, err
	}

	for _, eip := range eips.Items {
		clone := eip.DeepCopy()
		addr := args.unAssignIPFromEip(clone, peek)
		i.updateMetrics(clone)
		if addr != "" {
			if !reflect.DeepEqual(clone, eip) && !peek {
				err = i.Client.Status().Update(context.Background(), clone)
				ctrl.Log.Info("unAssignIP update eip", "eip", clone.Status)
			}

			result.Addr = addr
			result.Eip = eip.Name
			result.Protocol = eip.GetProtocol()
			result.Sp = speaker.GetSpeaker(eip.GetSpeakerName())

			if result.Sp == nil {
				err = fmt.Errorf("layer2 eip speaker not ready")
			}
			break
		}
	}

	i.log.Info("unAssignIP",
		"args", args,
		"peek", peek,
		"result", result,
		"err", err)

	return result, err
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
