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
	"flag"
	"net/http"
	"os"

	networkv1alpha1 "github.com/kubesphere/porter/api/v1alpha1"
	"github.com/kubesphere/porter/controllers/lb"
	bgpserver "github.com/kubesphere/porter/pkg/bgp/serverd"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var bgpStartOption *bgpserver.StartOption
var metricsAddr string
var readinessProbe bool
var enableLeaderElection bool

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = corev1.AddToScheme(scheme)
	_ = networkv1alpha1.AddToScheme(scheme)

	bgpStartOption = new(bgpserver.StartOption)
	flag.StringVar(&bgpStartOption.ConfigFile, "f", "", "specifying a config file,required")
	flag.StringVar(&bgpStartOption.ConfigType, "t", "toml", "specifying config type (toml, yaml, json)")
	flag.StringVar(&bgpStartOption.GrpcHosts, "api-hosts", ":50051", "specify the hosts that gobgpd listens on")
	flag.BoolVar(&bgpStartOption.GracefulRestart, "r", false, "flag restart-state in graceful-restart capability")
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")

}
func main() {
	flag.Parse()
	ctrl.SetLogger(zap.Logger(true))

	//starting bgp server
	setupLog.Info("starting bgp server")
	ready := make(chan interface{})
	go bgpserver.Run(bgpStartOption, ready)
	<-ready
	setupLog.Info("bgp server started successfully")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup all Controllers
	setupLog.Info("Setting up controller")
	if err = (&lb.ServiceReconciler{
		Client:        mgr.GetClient(),
		Log:           ctrl.Log.WithName("controllers").WithName("lb"),
		EventRecorder: mgr.GetEventRecorderFor("service"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "lb")
		os.Exit(1)
	}
	setupLog.Info("Setting up readiness probe")
	serverMuxA := http.NewServeMux()
	serverMuxA.HandleFunc("/hello", serveReadinessHandler)
	go func() {
		err := http.ListenAndServe(":8000", serverMuxA)
		if err != nil {
			setupLog.Error(err, "Failed to start readiness probe")
			os.Exit(1)
		}
	}()
	// Start the Cmd
	setupLog.Info("Starting the Cmd.")
	readinessProbe = true
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "unable to run the manager")
		os.Exit(1)
	}
}

func serveReadinessHandler(w http.ResponseWriter, r *http.Request) {
	if readinessProbe {
		w.WriteHeader(200)
		w.Write([]byte("Hello, World"))
	} else {
		w.WriteHeader(500)
		w.Write([]byte("Not Ready"))
	}
}
