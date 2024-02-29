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

	"github.com/openelb/openelb/cmd/manager/app"
	"github.com/openelb/openelb/cmd/manager/app/options"
	"github.com/openelb/openelb/pkg/log"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {

	s := options.NewOpenELBManagerOptions()

	log.InitLog(s.LogOptions)
	setupLog := ctrl.Log.WithName("setup")

	setupLog.Info("setting up openelb manager")

	command := app.NewOpenELBManagerCommand(s)

	if err := command.Execute(); err != nil {
		setupLog.Error(err, "unable to start openelb manager")
		os.Exit(1)
	}
}
