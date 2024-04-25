package ipam

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	networkv1alpha2 "github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/util"
	cnet "github.com/openelb/openelb/pkg/util/net"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type EIPController struct {
	client.Client
	log logr.Logger
	record.EventRecorder
}

const name = "EIPController"

func SetupWithManager(mgr ctrl.Manager) error {
	reconcile := &EIPController{
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
	}).Complete(reconcile)
}

// +kubebuilder:rbac:groups=network.kubesphere.io,resources=eips,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.kubesphere.io,resources=eips/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

func (i *EIPController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	i.log.Info("start setup openelb eip")
	defer i.log.Info("finish reconcile openelb eip")

	eip := &networkv1alpha2.Eip{}
	err := i.Get(ctx, req.NamespacedName, eip)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if util.IsDeletionCandidate(eip, constant.IPAMFinalizerName) {
		if err := i.removeEip(ctx, eip); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(eip, constant.IPAMFinalizerName)
		return ctrl.Result{}, i.Update(ctx, eip)
	}

	if util.NeedToAddFinalizer(eip, constant.IPAMFinalizerName) {
		controllerutil.AddFinalizer(eip, constant.IPAMFinalizerName)
		if err := i.Update(ctx, eip); err != nil {
			return ctrl.Result{}, err
		}
	}

	clone := eip.DeepCopy()
	if err = i.updateEip(ctx, clone); err != nil {
		i.Event(eip, v1.EventTypeWarning, EipAddOrUpdateReason, fmt.Sprintf("%s: %s", util.GetNodeName(), err.Error()))
		return ctrl.Result{}, err
	}

	if reflect.DeepEqual(clone.Status, eip.Status) {
		return ctrl.Result{}, nil
	}
	//i.updateMetrics(eip)
	return ctrl.Result{}, i.Client.Status().Update(ctx, clone)
}

func (i *EIPController) updateEip(ctx context.Context, e *networkv1alpha2.Eip) error {
	if e.Status.FirstIP == "" {
		base, size, _ := e.GetSize()
		e.Status.PoolSize = int(size)
		e.Status.FirstIP = base.String()
		e.Status.LastIP = cnet.IncrementIP(cnet.IP{IP: base}, big.NewInt(size-1)).String()
		if base.To4() != nil {
			e.Status.V4 = true
		}
	}

	return i.syncEip(ctx, e)
}

func (i *EIPController) syncEip(ctx context.Context, e *networkv1alpha2.Eip) error {
	used := make(map[string]string)
	for k, v := range e.Status.Used {
		svcs := strings.Split(v, ";")
		syncSvcs := []string{}
		for _, svc := range svcs {
			strs := strings.Split(svc, "/")
			if len(strs) < 2 {
				continue
			}

			obj := v1.Service{}
			err := i.Get(ctx, client.ObjectKey{Namespace: strs[0], Name: strs[1]}, &obj)
			if err != nil {
				if errors.IsNotFound(err) {
					continue
				}
				return err
			}

			syncSvcs = append(syncSvcs, svc)
		}

		if len(syncSvcs) != 0 {
			used[k] = strings.Join(syncSvcs, ";")
		}
	}
	e.Status.Used = used
	e.Status.Usage = len(used)
	e.Status.Occupied = e.Status.Usage >= e.Status.PoolSize

	return nil
}

func (i *EIPController) removeEip(ctx context.Context, e *networkv1alpha2.Eip) error {
	svcs := v1.ServiceList{}
	opts := labels.SelectorFromSet(labels.Set(map[string]string{constant.OpenELBEIPAnnotationKeyV1Alpha2: e.Name}))
	err := i.List(ctx, &svcs, &client.ListOptions{LabelSelector: opts})
	if err != nil {
		return err
	}

	for _, svc := range svcs.Items {
		delete(svc.Labels, constant.OpenELBEIPAnnotationKeyV1Alpha2)
		if err := i.Update(ctx, &svc); err != nil {
			return err
		}
	}

	return nil
}
