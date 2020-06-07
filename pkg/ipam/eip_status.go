package ipam

import (
	"context"
	"reflect"
	"time"

	"github.com/kubesphere/porter/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

const (
	DefaultSyncInterval = time.Second * 10
)

func (i *IPAM) Start(stop <-chan struct{}) error {
	i.log.Info("Starting EIP Updater")
	for {
		select {
		case <-stop:
			i.log.Info("Recieve stop signal, stopping")
			return nil
		case <-time.After(i.syncInterval):
			i.do()
		}
	}
}

func (i *IPAM) do() {
	eiplist := &v1alpha1.EipList{}
	err := i.client.List(context.TODO(), eiplist)
	if err != nil {
		i.log.Error(err, "Failed to list eip, waiting for next try")
		return
	}
	for _, eip := range eiplist.Items {
		err = i.syncEIP(&eip)
		if err != nil {
			i.log.Error(err, "Failed to sync eip, waiting for next try", "Name", eip.Name, "CIDR", eip.Spec.Address)
		}
	}
}

func (i *IPAM) syncEIP(eip *v1alpha1.Eip) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		original := &v1alpha1.Eip{}
		err := i.client.Get(context.TODO(), types.NamespacedName{Name: eip.Name}, original)
		if err != nil {
			return err
		}
		if !original.ObjectMeta.DeletionTimestamp.IsZero() {
			i.log.V(2).Info("eip is deleting, skipping", "eip", eip.Spec.Address)
			return nil
		}
		status, err := i.ds.GetPoolUsage(eip.Name)
		if err != nil {
			return err
		}
		if reflect.DeepEqual(original.Status, status) {
			return nil
		}
		original.Status = status
		i.log.Info("current eip usage", "use", status.Usage, "total", status.PoolSize, "eip", original.Spec.Address)
		return i.client.Status().Update(context.TODO(), original)
	})
}
