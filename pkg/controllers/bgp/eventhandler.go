package bgp

import (
	"context"
	"github.com/openelb/openelb/api/v1alpha2"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var nodeEnqueueLog = ctrl.Log.WithName("eventhandler").WithName("EnqueueRequestForNode")

type EnqueueRequestForNode struct {
	client.Client
	peer bool
}

func (e *EnqueueRequestForNode) getDefaultBgpConf() []v1alpha2.BgpConf {
	var def v1alpha2.BgpConf

	if err := e.Get(context.Background(), client.ObjectKey{Name: "default"}, &def); err != nil {
		return nil
	}

	return []v1alpha2.BgpConf{def}
}

func (e *EnqueueRequestForNode) getBgpPeers() []v1alpha2.BgpPeer {
	var peers v1alpha2.BgpPeerList

	if err := e.List(context.Background(), &peers); err != nil {
		return nil
	}

	return peers.Items
}

// Create implements EventHandler
func (e *EnqueueRequestForNode) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
}

// Update implements EventHandler
func (e *EnqueueRequestForNode) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.MetaOld == nil {
		nodeEnqueueLog.Error(nil, "UpdateEvent received with no old metadata", "event", evt)
	}

	if evt.MetaNew == nil {
		nodeEnqueueLog.Error(nil, "UpdateEvent received with no new metadata", "event", evt)
	}

	if !e.peer {
		for _, svc := range e.getDefaultBgpConf() {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      svc.GetName(),
				Namespace: svc.GetNamespace(),
			}})
		}
	} else {
		for _, svc := range e.getBgpPeers() {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      svc.GetName(),
				Namespace: svc.GetNamespace(),
			}})
		}
	}
}

// Delete implements EventHandler
func (e *EnqueueRequestForNode) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
}

// Generic implements EventHandler
func (e *EnqueueRequestForNode) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {

}
