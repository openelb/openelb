package ipam

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/constant"
	"github.com/kubesphere/porter/pkg/util"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type IPAM struct {
	client client.Client
	log    logr.Logger

	ds *DataStore

	syncInterval time.Duration
}

func NewIPAM(log logr.Logger, ds *DataStore) *IPAM {
	return &IPAM{
		log:          log,
		ds:           ds,
		syncInterval: DefaultSyncInterval,
	}
}

func (i *IPAM) SetupWithManager(mgr manager.Manager) error {
	i.client = mgr.GetClient()

	i.log.Info("Setting up EIPUpdater")
	if err := mgr.Add(i); err != nil {
		i.log.Error(nil, "Failed to run EIPUpdater")
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).Named("IPAM").
		For(&networkv1alpha1.Eip{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				//only support create and delete, because change EIP will affect a lot
				//so we should make sure no service use EIP, and then delete , create a new one.
				if !e.MetaNew.GetDeletionTimestamp().IsZero() {
					return true
				}
				return false
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

	deleted, err := i.useFinalizerIfNeeded(eip)
	if err != nil {
		return ctrl.Result{}, err
	}
	if deleted {
		return ctrl.Result{}, nil
	}

	err = i.ds.AddEIPPool(eip.Spec.Address, eip.Name, eip.Spec.UsingKnownIPs, eip.Spec.Protocol)
	return ctrl.Result{RequeueAfter: time.Second * 60}, err
}

func (i *IPAM) useFinalizerIfNeeded(eip *networkv1alpha1.Eip) (bool, error) {
	i.log.Info("handling finalizer")
	if eip.ObjectMeta.DeletionTimestamp.IsZero() {
		if !util.ContainsString(eip.ObjectMeta.Finalizers, constant.IPAMFinalizerName) {
			eip.ObjectMeta.Finalizers = append(eip.ObjectMeta.Finalizers, constant.IPAMFinalizerName)
			if err := i.client.Update(context.Background(), eip); err != nil {
				return false, err
			}
			i.log.Info("Append Finalizer to eip")
			return false, nil
		}
	} else {
		// The object is being deleted
		if util.ContainsString(eip.ObjectMeta.Finalizers, constant.IPAMFinalizerName) {
			i.log.Info("Begin to remove finalizer")
			if err := i.ds.RemoveEIPPool(eip.Spec.Address, eip.Name); err != nil {
				return false, err
			}
			eip.ObjectMeta.Finalizers = util.RemoveString(eip.ObjectMeta.Finalizers, constant.IPAMFinalizerName)
			if err := i.client.Update(context.Background(), eip); err != nil {
				if k8serrors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			}
			i.log.Info("Remove Finalizer before eip deleted")
		}
		return true, nil
	}
	return false, nil
}
