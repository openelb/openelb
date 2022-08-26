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
	"reflect"
	"time"

	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/speaker/bgp"
	"github.com/openelb/openelb/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	syncStatusPeriod = 30
	policyField      = ".spec.policy"
)

// BgpConfReconciler reconciles a BgpConf object
type BgpConfReconciler struct {
	client.Client
	BgpServer *bgp.Bgp
	record.EventRecorder
	cleaned bool
}

//+kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=network.kubesphere.io,resources=bgpconfs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster BgpConf CRD closer to the desired state.
func (r *BgpConfReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	instance := &v1alpha2.BgpConf{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return. Created objects are automatically garbage collected.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	clone := instance.DeepCopy()

	cm, err := r.getPolicyConfigMap(ctx, clone)
	if err != nil {
		return ctrl.Result{}, err
	}

	if util.IsDeletionCandidate(clone, constant.FinalizerName) {
		err := r.BgpServer.HandleBgpGlobalConfig(clone, "", true, nil)
		if err != nil {
			ctrl.Log.Error(err, "cannot delete bgp conf, maybe need to delete manually")
		}

		controllerutil.RemoveFinalizer(clone, constant.FinalizerName)
		return ctrl.Result{}, r.Update(context.Background(), clone)
	}

	if util.NeedToAddFinalizer(clone, constant.FinalizerName) {
		controllerutil.AddFinalizer(clone, constant.FinalizerName)
		err := r.Update(context.Background(), clone)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	node := &corev1.Node{}
	rack := ""
	err = r.Get(context.Background(), types.NamespacedName{Name: util.GetNodeName()}, node)
	if err != nil {
		return ctrl.Result{}, err
	}
	if node.Labels != nil && node.Labels[constant.OpenELBNodeRack] != "" && clone.Spec.AsPerRack != nil {
		rack = node.Labels[constant.OpenELBNodeRack]
		as := clone.Spec.AsPerRack[rack]
		if as > 0 {
			clone.Spec.As = as
		}
		clone.Spec.RouterId = ""
	}
	if clone.Spec.RouterId == "" {
		clone.Spec.RouterId = util.GetNodeIP(*node).String()
	}

	err = r.BgpServer.HandleBgpGlobalConfig(clone, rack, false, cm)
	if err != nil {
		return ctrl.Result{}, err
	}

	if clone.Annotations == nil {
		clone.Annotations = make(map[string]string)
	}
	clone.Spec = instance.Spec
	clone.Annotations[constant.OpenELBAnnotationKey] = time.Now().String()
	err = r.Client.Update(context.Background(), clone)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, r.reconfigPeers()
}

func (r *BgpConfReconciler) getPolicyConfigMap(ctx context.Context, bgpConf *v1alpha2.BgpConf) (*corev1.ConfigMap, error) {
	if bgpConf.Spec.Policy == "" {
		return nil, nil
	}
	policyName := bgpConf.Spec.Policy
	foundPolicy := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: policyName, Namespace: util.EnvNamespace()}, foundPolicy)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return foundPolicy, nil
}

func (r *BgpConfReconciler) Map(configMap handler.MapObject) []reconcile.Request {
	attachedBgpConfs := &v1alpha2.BgpConfList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(policyField, configMap.Meta.GetName()),
	}
	err := r.List(context.TODO(), attachedBgpConfs, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedBgpConfs.Items))
	for i, item := range attachedBgpConfs.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: item.GetName(),
			},
		}
	}
	return requests
}

func (r *BgpConfReconciler) reconfigPeers() error {
	ctx := context.Background()

	//Add all the neighbor that exist and match node back in, since
	//the neighbor was reset when the global configuration was updated earlier.
	var peers v1alpha2.BgpPeerList
	err := r.List(ctx, &peers)
	if err != nil {
		return err
	}
	node := &corev1.Node{}
	err = r.Get(ctx, types.NamespacedName{Name: util.GetNodeName()}, node)
	if err != nil {
		return err
	}
	for _, peer := range peers.Items {
		match, err := peerMatchNode(&peer, node)
		if err != nil {
			return err
		}
		if match {
			err = r.BgpServer.HandleBgpPeer(&peer, false)
			if err != nil {
				return err
			}
		}
	}

	return nil
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
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			if util.DutyOfCNI(nil, e.Meta) {
				return false
			}
			return shouldReconcile(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if shouldReconcile(e.ObjectNew) {
				oldConf := e.ObjectOld.(*v1alpha2.BgpConf)
				newConf := e.ObjectNew.(*v1alpha2.BgpConf)

				if !util.DutyOfCNI(e.MetaOld, e.MetaNew) {
					if !reflect.DeepEqual(oldConf.DeletionTimestamp, newConf.DeletionTimestamp) {
						return true
					}

					if !reflect.DeepEqual(oldConf.Spec, newConf.Spec) {
						return true
					}
				}
			}

			return false
		},
	}

	// The policy field must be indexed by the manager, so that we will be able to lookup BgpConf by a referenced ConfigMap name.
	err := mgr.GetFieldIndexer().
		IndexField(context.Background(), &v1alpha2.BgpConf{}, policyField, func(rawObj runtime.Object) []string {
			// Extract the ConfigMap name from the BgpConf Spec, if one is provided
			bgpConf := rawObj.(*v1alpha2.BgpConf)
			if bgpConf.Spec.Policy == "" {
				return nil
			}
			return []string{bgpConf.Spec.Policy}
		})
	if err != nil {
		return err
	}

	ctl, err := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha2.BgpConf{}, builder.WithPredicates(p)).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: r},
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Named("BgpConfController").
		Build(r)
	if err != nil {
		return err
	}

	np := predicate.Funcs{
		UpdateFunc: func(evt event.UpdateEvent) bool {
			old := evt.ObjectOld.(*corev1.Node)
			new := evt.ObjectNew.(*corev1.Node)

			oldHaveLabel := false
			if old.Labels != nil {
				_, oldHaveLabel = old.Labels[constant.OpenELBNodeRack]
			}
			newHaveLabel := false
			if new.Labels != nil {
				_, newHaveLabel = old.Labels[constant.OpenELBNodeRack]
			}
			if oldHaveLabel != newHaveLabel {
				return true
			}

			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
	}
	return ctl.Watch(&source.Kind{Type: &corev1.Node{}}, &EnqueueRequestForNode{Client: r.Client, peer: false}, np)
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

	return mgr.Add(bgpConf)
}

func (r *BgpConfReconciler) CleanBgpConfStatus() error {
	instance := &v1alpha2.BgpConf{}
	err := r.Client.Get(context.Background(), client.ObjectKey{Name: "default"}, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	clone := instance.DeepCopy()
	clone.Status = v1alpha2.BgpConfStatus{}
	if reflect.DeepEqual(clone.Status, instance.Status) {
		return nil
	}
	return r.Client.Status().Update(context.Background(), clone)
}

func (r BgpConfReconciler) updateConfStatus() {
	instance := &v1alpha2.BgpConf{}
	err := r.Client.Get(context.Background(), client.ObjectKey{Name: "default"}, instance)
	if err != nil {
		return
	}
	clone := instance.DeepCopy()
	if clone.Status.NodesConfStatus == nil {
		clone.Status.NodesConfStatus = make(map[string]v1alpha2.NodeConfStatus)
	}
	result := r.BgpServer.GetBgpConfStatus()
	nodeName := util.GetNodeName()
	clone.Status.NodesConfStatus[nodeName] = result.Status.NodesConfStatus[nodeName]

	if reflect.DeepEqual(clone.Status, instance.Status) {
		return
	}
	r.Client.Status().Update(context.Background(), clone)
}

func (r BgpConfReconciler) run(stopCh <-chan struct{}) {
	t := time.NewTicker(time.Duration(syncStatusPeriod) * time.Second)

	for {
		select {
		case <-t.C:
			r.updateConfStatus()

		case <-stopCh:
			return
		}
	}
}

func (r BgpConfReconciler) Start(stopCh <-chan struct{}) error {
	err := r.CleanBgpConfStatus()
	if err != nil {
		return err
	}

	go r.run(stopCh)

	return nil
}
