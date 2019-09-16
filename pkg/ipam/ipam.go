package ipam

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/util/retry"

	"github.com/go-logr/logr"
	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/constant"
	"github.com/kubesphere/porter/pkg/errors"
	"github.com/kubesphere/porter/pkg/util"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	DefaultSyncInterval = time.Second * 10
)

type IPAM struct {
	client       client.Client
	Log          logr.Logger
	ds           *DataStore
	SyncInterval time.Duration
	EIPUpdater   *EIPUpdater
}

func (i *IPAM) CheckEIPStatus(eip string) *EIPStatus {
	return i.ds.GetEIPStatus(eip)
}

func NewIPAM(log logr.Logger) *IPAM {
	return &IPAM{
		Log: log,
		ds:  NewDataStore(log),
	}
}

func (i *IPAM) SetupWithManager(mgr manager.Manager) error {
	i.client = mgr.GetClient()
	if i.SyncInterval == 0 {
		i.SyncInterval = DefaultSyncInterval
	}
	i.EIPUpdater = NewEIPUpdaterFromIPAM(i)
	i.Log.Info("Setting up EIPUpdater")
	if err := mgr.Add(i.EIPUpdater); err != nil {
		i.Log.Error(nil, "Failed to run EIPUpdater")
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).Named("IPAM").
		For(&networkv1alpha1.Eip{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				old := e.ObjectOld.(*networkv1alpha1.Eip)
				new := e.ObjectNew.(*networkv1alpha1.Eip)
				if !e.MetaNew.GetDeletionTimestamp().IsZero() {
					return true
				}
				return old.Spec.Address != new.Spec.Address
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
		}).
		Complete(i)
}

func (i *IPAM) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	eip := &networkv1alpha1.Eip{}
	err := i.client.Get(context.TODO(), req.NamespacedName, eip)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	_ = i.Log.WithValues("name", eip.Name, "address", eip.Spec.Address)
	var deleted bool
	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		err := i.client.Get(context.TODO(), types.NamespacedName{Name: eip.Name}, eip)
		if err != nil {
			return err
		}
		deleted, err = i.useFinalizerIfNeeded(eip)
		return err
	})
	if err != nil {
		i.Log.Error(nil, "Failed to handle finalizer")
		return ctrl.Result{}, err
	}
	if deleted {
		return ctrl.Result{}, nil
	}
	err = i.addEIPtoDataStore(eip)
	if err != nil {
		if _, ok := err.(errors.DataStoreEIPDuplicateError); ok {
			i.Log.Info("Detect this eip is in pool now, skipping")
			return ctrl.Result{}, nil
		}
		i.Log.Error(err, "could not add eip to pool")
		return ctrl.Result{RequeueAfter: time.Second * 10}, nil
	}
	return ctrl.Result{}, nil
}

func (i *IPAM) addEIPtoDataStore(eip *networkv1alpha1.Eip) error {
	i.Log.Info("Add EIP to pool")
	return i.ds.AddEIPPool(eip.Spec.Address, eip.Name, eip.Spec.UsingKnownIPs)
}

func (i *IPAM) useFinalizerIfNeeded(eip *networkv1alpha1.Eip) (bool, error) {
	i.Log.Info("handling finalizer")
	if eip.ObjectMeta.DeletionTimestamp.IsZero() {
		if !util.ContainsString(eip.ObjectMeta.Finalizers, constant.IPAMFinalizerName) {
			eip.ObjectMeta.Finalizers = append(eip.ObjectMeta.Finalizers, constant.IPAMFinalizerName)
			if err := i.client.Update(context.Background(), eip); err != nil {
				return false, err
			}
			i.Log.Info("Append Finalizer to eip")
			return false, nil
		}
	} else {
		// The object is being deleted
		if util.ContainsString(eip.ObjectMeta.Finalizers, constant.IPAMFinalizerName) {
			i.Log.Info("Begin to remove finalizer")
			if err := i.ds.RemoveEIPPool(eip.Spec.Address, eip.Name); err != nil {
				if _, ok := err.(errors.DataStoreEIPNotExist); ok {
					i.Log.Info("EIP is no longer in pool", "eip", eip.Spec.Address)
				} else {
					i.Log.Error(nil, "Failed to remove eip from pool")
					return false, err
				}
			}
			eip.ObjectMeta.Finalizers = util.RemoveString(eip.ObjectMeta.Finalizers, constant.IPAMFinalizerName)
			if err := i.client.Update(context.Background(), eip); err != nil {
				if k8serrors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			i.Log.Info("Remove Finalizer before eip deleted")
		}
		return true, nil
	}
	return false, nil
}

func (i *IPAM) AssignIP(serv *corev1.Service) (*AssignIPResponse, error) {
	return i.ds.AssignIP(serv.Name, serv.Namespace)
}

func (i *IPAM) RevokeIP(ip string) error {
	return i.ds.UnassignIP(ip)
}

func (i *IPAM) AssignSpecifyIP(serv *corev1.Service, ip string) (*AssignIPResponse, error) {
	return i.ds.AssignSpecifyIP(ip, serv.Name, serv.Namespace)
}
