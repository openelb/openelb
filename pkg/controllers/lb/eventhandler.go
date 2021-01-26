package lb

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
}

func (e *EnqueueRequestForNode) getServices() []corev1.Service {
	var svcs corev1.ServiceList

	if err := e.List(context.Background(), &svcs); err != nil {
		nodeEnqueueLog.Error(err, "Failed to list services")
		return nil
	}

	var result []corev1.Service
	for _, svc := range svcs.Items {
		if IsPorterService(&svc) {
			result = append(result, svc)
		}
	}

	return result
}

// Create implements EventHandler
func (e *EnqueueRequestForNode) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Meta == nil {
		nodeEnqueueLog.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}

	for _, svc := range e.getServices() {
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      svc.GetName(),
			Namespace: svc.GetNamespace(),
		}})
	}
}

func nodeReady(obj runtime.Object) bool {
	node := obj.(*corev1.Node)

	ready := true
	for _, con := range node.Status.Conditions {
		if con.Type == corev1.NodeReady && con.Status != corev1.ConditionTrue {
			ready = false
			break
		}
		if con.Type == corev1.NodeNetworkUnavailable && con.Status != corev1.ConditionFalse {
			ready = false
			break
		}
	}

	return ready
}

// Update implements EventHandler
func (e *EnqueueRequestForNode) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.MetaOld == nil {
		nodeEnqueueLog.Error(nil, "UpdateEvent received with no old metadata", "event", evt)
	}

	if evt.MetaNew == nil {
		nodeEnqueueLog.Error(nil, "UpdateEvent received with no new metadata", "event", evt)
	}

	for _, svc := range e.getServices() {
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      svc.GetName(),
			Namespace: svc.GetNamespace(),
		}})
	}
}

// Delete implements EventHandler
func (e *EnqueueRequestForNode) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Meta == nil {
		nodeEnqueueLog.Error(nil, "DeleteEvent received with no metadata", "event", evt)
		return
	}
	for _, svc := range e.getServices() {
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      svc.GetName(),
			Namespace: svc.GetNamespace(),
		}})
	}
}

// Generic implements EventHandler
func (e *EnqueueRequestForNode) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {

}
