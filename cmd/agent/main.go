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

package main

import (
	"os"

	"github.com/openelb/openelb/pkg/log"

	networkv1alpha2 "github.com/openelb/openelb/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = networkv1alpha2.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
}

func main() {
	log.InitLog(log.NewOptions())

	setupLog := ctrl.Log.WithName("setup")

	setupLog.Info("setting up openelb agent")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		setupLog.Error(err, "unable to start openelb agent")
		os.Exit(1)
	}

	mgr.Add(Fake{})

	// Start the Cmd
	setupLog.Info("Starting the openelb agent")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "unable to run the openelb agent")
		os.Exit(1)
	}
}

// At the moment, the agent has no tasks to do, but it may
//be needed for future extensions, so it is kept here.
type Fake struct {
}

func (f Fake) Start(stopCh <-chan struct{}) error {
	<-stopCh
	return nil
}
