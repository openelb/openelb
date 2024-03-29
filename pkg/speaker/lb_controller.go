/*
Copyright 2020 The Kubesphere Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package speaker

import (
	"context"
	"time"

	"github.com/openelb/openelb/pkg/constant"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// LBReconciler reconciles a service object
type LBReconciler struct {
	client.Client
	record.EventRecorder

	Handler func(context.Context, *corev1.Service) error
}

func IsOpenELBService(svc *corev1.Service) bool {
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return false
	}

	return svc.Annotations[constant.OpenELBAnnotationKey] == constant.OpenELBAnnotationValue
}

func (r *LBReconciler) shouldReconcileEP(e metav1.Object) bool {
	if e.GetAnnotations()["control-plane.alpha.kubernetes.io/leader"] != "" {
		return false
	}

	svc := &corev1.Service{}
	err := r.Get(context.TODO(), types.NamespacedName{Namespace: e.GetNamespace(), Name: e.GetName()}, svc)
	if err != nil {
		return !errors.IsNotFound(err)
	}

	if svc.Spec.ExternalTrafficPolicy == corev1.ServiceExternalTrafficPolicyTypeLocal {
		return IsOpenELBService(svc)
	}
	return false
}

func (r *LBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			old, ok := e.ObjectOld.(*corev1.Service)
			if !ok {
				return false
			}

			new, ok := e.ObjectNew.(*corev1.Service)
			if !ok {
				return false
			}

			if old.Spec.ExternalTrafficPolicy == new.Spec.ExternalTrafficPolicy {
				return false
			}

			return IsOpenELBService(new)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}

	ctl, err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Owns(&corev1.Endpoints{}).
		WithEventFilter(p).
		Named("LBController").
		Build(r)
	if err != nil {
		return err
	}

	//endpoints
	ep := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return r.shouldReconcileEP(e.ObjectNew)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return r.shouldReconcileEP(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.shouldReconcileEP(e.Object)
		},
	}

	return ctl.Watch(source.Kind(mgr.GetCache(), &corev1.Endpoints{}), &handler.EnqueueRequestForObject{}, ep)
}

//+kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster BgpConf CRD closer to the desired state.
func (l *LBReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.V(4).Infof("Starting to sync service %s/%s", req.Namespace, req.Name)
	startTime := time.Now()

	defer func() {
		klog.V(4).Infof("Finished syncing service %s/%s in %s", req.Namespace, req.Name, time.Since(startTime))
	}()

	svc := &corev1.Service{}
	if err := l.Client.Get(ctx, req.NamespacedName, svc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, l.Handler(ctx, svc)
}
