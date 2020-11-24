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

package bgp

import (
	"context"
	"fmt"
	"reflect"

	"github.com/kubesphere/porter/api/v1alpha2"
	"github.com/kubesphere/porter/pkg/constant"
	"github.com/kubesphere/porter/pkg/speaker/bgp"
	"github.com/kubesphere/porter/pkg/util"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// BgpConfReconciler reconciles a BgpConf object
type BgpConfReconciler struct {
	client.Client
	BgpServer *bgp.Bgp
	record.EventRecorder
}

// +kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs/status,verbs=get;update;patch

func (r *BgpConfReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	instance := &v1alpha2.BgpConf{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	clone := instance.DeepCopy()

	if util.IsDeletionCandidate(clone, constant.FinalizerName) {
		err := r.BgpServer.HandleBgpGlobalConfig(clone, true)
		if err != nil {
			ctrl.Log.Error(err, "cannot delete bgp conf, maybe need to delete manually")
		}

		controllerutil.RemoveFinalizer(clone, constant.FinalizerName)
		err = r.Update(context.Background(), clone)
		return ctrl.Result{}, err
	}

	if util.NeedToAddFinalizer(clone, constant.FinalizerName) {
		controllerutil.AddFinalizer(clone, constant.FinalizerName)
		err := r.Update(context.Background(), clone)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if clone.Spec.RouterId == "" {
		clone.Spec.RouterId, err = r.getRouterID()
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	if clone.Status.NodesConfStatus == nil {
		clone.Status.NodesConfStatus = make(map[string]v1alpha2.NodeConfStatus)
	}
	clone.Status.NodesConfStatus[util.GetNodeName()] = v1alpha2.NodeConfStatus{
		RouterId: clone.Spec.RouterId,
	}

	err = r.BgpServer.HandleBgpGlobalConfig(clone, false)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(clone.Status, instance.Status) {
		err = r.Client.Status().Update(context.Background(), clone)
	}

	return ctrl.Result{}, err
}

func (r *BgpConfReconciler) getRouterID() (string, error) {
	var nodes v1.NodeList

	err := r.List(context.Background(), &nodes)
	if err != nil {
		return "", err
	}

	nodeName := util.GetNodeName()

	for _, node := range nodes.Items {
		if node.Name == nodeName {
			ip := util.GetNodeIP(node)
			if ip == nil {
				return "", fmt.Errorf("%s: has no valild ip", nodeName)
			}
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("nodename %s not match", nodeName)
}

func shouldReconcile(obj runtime.Object) bool {
	if conf, ok := obj.(*v1alpha2.BgpConf); ok {
		if conf.Name == "default" {
			return true
		}
	}

	return false
}

func (r *BgpConfReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.BgpConf{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return shouldReconcile(e.Object)
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				if shouldReconcile(e.ObjectNew) {
					old := e.ObjectOld.(*v1alpha2.BgpConf)
					new := e.ObjectNew.(*v1alpha2.BgpConf)
					if !reflect.DeepEqual(old.DeletionTimestamp, new.DeletionTimestamp) {
						return true
					}

					if !reflect.DeepEqual(old.Spec, new.Spec) {
						return true
					}
				}

				return false
			},
		}).Complete(r)
}

func SetupBgpConfReconciler(bgpServer *bgp.Bgp, mgr ctrl.Manager) error {
	bgpConf := BgpConfReconciler{
		Client:        mgr.GetClient(),
		BgpServer:     bgpServer,
		EventRecorder: mgr.GetEventRecorderFor("bgpconf"),
	}
	if err := bgpConf.SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}
