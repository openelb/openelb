/*
Copyright 2019 The Kubesphere Authors.

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
package service

import (
	"context"

	"github.com/kubesphere/porter/pkg/validate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type WatchFunc func(*WatchManager) error
type WatchManager struct {
	manager    manager.Manager
	controller controller.Controller
	fns        []WatchFunc
}

func (c *WatchManager) AddAllWatch() error {
	for _, fn := range c.fns {
		err := fn(c)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewWatchManager(c controller.Controller, m manager.Manager) *WatchManager {
	wm := &WatchManager{
		manager:    m,
		controller: c,
		fns:        make([]WatchFunc, 0),
	}
	wm.fns = append(wm.fns, watchService)
	wm.fns = append(wm.fns, watchEndPoint)
	return wm
}
func watchService(c *WatchManager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if validate.IsTypeLoadBalancer(e.ObjectOld) || validate.IsTypeLoadBalancer(e.ObjectNew) {
				if validate.HasPorterLBAnnotation(e.MetaNew.GetAnnotations()) || validate.HasPorterLBAnnotation(e.MetaOld.GetAnnotations()) {
					return e.ObjectOld != e.ObjectNew
				}
			}
			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			if validate.IsTypeLoadBalancer(e.Object) {
				return validate.HasPorterLBAnnotation(e.Meta.GetAnnotations())
			}
			return false
		},
	}
	// Watch for changes to Service
	return c.controller.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForObject{}, p)
}

func watchEndPoint(c *WatchManager) error {
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			svc := &corev1.Service{}
			err := c.manager.GetClient().Get(context.TODO(), types.NamespacedName{Namespace: e.MetaOld.GetNamespace(), Name: e.MetaOld.GetName()}, svc)
			if err != nil {
				return false
			}
			if validate.IsTypeLoadBalancer(svc) && validate.HasPorterLBAnnotation(svc.GetAnnotations()) {
				old := e.ObjectOld.(*corev1.Endpoints)
				new := e.ObjectNew.(*corev1.Endpoints)
				return validate.IsNodeChangeWhenEPUpdate(old, new)
			}
			return false
		},
		CreateFunc: func(e event.CreateEvent) bool {
			svc := &corev1.Service{}
			err := c.manager.GetClient().Get(context.TODO(), types.NamespacedName{Namespace: e.Meta.GetNamespace(), Name: e.Meta.GetName()}, svc)
			if err != nil {
				return false
			}
			if validate.IsTypeLoadBalancer(svc) {
				return validate.HasPorterLBAnnotation(svc.GetAnnotations())
			}
			return false
		},
	}
	return c.controller.Watch(&source.Kind{Type: &corev1.Endpoints{}}, &handler.EnqueueRequestForObject{}, p)
}
