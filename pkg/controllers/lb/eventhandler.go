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
		if IsOpenELBService(&svc) {
			result = append(result, svc)
		}
	}

	return result
}

// Create implements EventHandler
func (e *EnqueueRequestForNode) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
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

// When Node addr changed, system should update all OpenELB services
func nodeAddrChange(oldObj runtime.Object, newObj runtime.Object) bool {
	addrChange := false

	oldExternalIP, oldInternalIP := nodeInternalAndExternalIP(oldObj)
	newExternalIP, newInternalIP := nodeInternalAndExternalIP(newObj)
	if oldExternalIP != newExternalIP {
		addrChange = true
	}
	if oldInternalIP != newInternalIP {
		addrChange = true
	}

	return addrChange
}

func nodeInternalAndExternalIP(obj runtime.Object) (externalIP, internalIP string) {
	node := obj.(*corev1.Node)

	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeExternalIP {
			externalIP = addr.Address
		}
		if addr.Type == corev1.NodeInternalIP {
			internalIP = addr.Address
		}
	}
	return
}

// Update implements EventHandler
func (e *EnqueueRequestForNode) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.ObjectOld == nil {
		nodeEnqueueLog.Error(nil, "UpdateEvent received with no old metadata", "event", evt)
	}

	if evt.ObjectNew == nil {
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
func (e *EnqueueRequestForNode) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
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
func (e *EnqueueRequestForNode) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {

}

var deAndDsEnqueueLog = ctrl.Log.WithName("eventhandler").WithName("EnqueueRequestForDeAndDs")

// Enqueue requests for Deployments and DaemonSets type
// Only OpenELB NodeProxy needs this
type EnqueueRequestForDeAndDs struct {
	client.Client
}

// Get all OpenELB NodeProxy Services to reconcile them later
// These Services will be exposed by Proxy Pod
func (e *EnqueueRequestForDeAndDs) getServices() []corev1.Service {
	var svcs corev1.ServiceList

	if err := e.List(context.Background(), &svcs); err != nil {
		deAndDsEnqueueLog.Error(err, "Failed to list services")
		return nil
	}

	var result []corev1.Service
	for _, svc := range svcs.Items {
		if IsOpenELBNPService(&svc) {
			result = append(result, svc)
		}
	}

	return result
}

// Create implements EventHandler
func (e *EnqueueRequestForDeAndDs) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		deAndDsEnqueueLog.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}

	for _, svc := range e.getServices() {
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      svc.GetName(),
			Namespace: svc.GetNamespace(),
		}})
	}
}

// Update implements EventHandler
func (e *EnqueueRequestForDeAndDs) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.ObjectOld == nil {
		deAndDsEnqueueLog.Error(nil, "UpdateEvent received with no old metadata", "event", evt)
	}

	if evt.ObjectNew == nil {
		deAndDsEnqueueLog.Error(nil, "UpdateEvent received with no new metadata", "event", evt)
	}

	for _, svc := range e.getServices() {
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      svc.GetName(),
			Namespace: svc.GetNamespace(),
		}})
	}
}

// Delete implements EventHandler
func (e *EnqueueRequestForDeAndDs) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Object == nil {
		deAndDsEnqueueLog.Error(nil, "DeleteEvent received with no metadata", "event", evt)
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
func (e *EnqueueRequestForDeAndDs) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {

}
