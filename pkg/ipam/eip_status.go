package ipam

import (
	"context"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	"github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/pkg/errors"
	"github.com/kubesphere/porter/pkg/util"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EIPUpdater struct {
	ds *DataStore
	client.Client
	log          logr.Logger
	syncInterval time.Duration
}

func NewEIPUpdaterFromIPAM(i *IPAM) *EIPUpdater {
	return &EIPUpdater{
		ds:           i.ds,
		Client:       i.client,
		syncInterval: i.SyncInterval,
		log:          i.Log.WithName("EIP_Updater"),
	}
}

func (e *EIPUpdater) Start(stop <-chan struct{}) error {
	e.log.Info("Starting EIP Updater")
	for {
		select {
		case <-stop:
			e.log.Info("Recieve stop signal, stopping")
			return nil
		default:
			e.do()
		}
	}
}

func (e *EIPUpdater) do() {
	e.log.Info("Begin to sync eiplist")
	time.Sleep(e.syncInterval)
	eiplist := &v1alpha1.EipList{}
	err := e.List(context.TODO(), eiplist)
	if err != nil {
		e.log.Error(err, "Failed to list eip, waiting for next try")
		return
	}
	for _, eip := range eiplist.Items {
		err = e.syncEIP(&eip)
		if err != nil {
			e.log.Error(err, "Failed to sync eip, waiting for next try", "Name", eip.Name, "CIDR", eip.Spec.Address)
		}
	}
	e.log.Info("Eiplist synced")
}

func (e *EIPUpdater) syncEIP(eip *v1alpha1.Eip) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		original := &v1alpha1.Eip{}
		err := e.Get(context.TODO(), types.NamespacedName{Name: eip.Name}, original)
		if err != nil {
			return err
		}
		if !original.ObjectMeta.DeletionTimestamp.IsZero() {
			e.log.V(2).Info("eip is deleting, skipping", "eip", eip.Spec.Address)
			return nil
		}
		instance := original.DeepCopy()
		pool, ok := e.ds.IPPool[eip.Name]
		if !ok {
			return errors.NewEIPNotFoundError(eip.Name)
		}
		if instance.Status.PoolSize == 0 {
			instance.Status.PoolSize = util.GetValidAddressCount(instance.Spec.Address)
		}
		instance.Status.Usage = len(pool.Used)
		if instance.Status.Usage == instance.Status.PoolSize {
			instance.Status.Occupied = true
		} else {
			instance.Status.Occupied = false
		}
		if reflect.DeepEqual(original.Status, instance.Status) {
			return nil
		}
		e.logEIPInfo(instance)
		return e.Status().Update(context.TODO(), instance)
	})
}

func (e *EIPUpdater) logEIPInfo(eip *v1alpha1.Eip) {
	e.log.Info("current eip usage", "use", eip.Status.Usage, "total", eip.Status.PoolSize, "eip", eip.Spec.Address)
}
