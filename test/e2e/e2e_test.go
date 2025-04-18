/*
Copyright 2015 The Kubernetes Authors.

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

package e2e

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/test/e2e"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/config"

	// test sources
	_ "github.com/openelb/openelb/test/e2e/bgp"
	_ "github.com/openelb/openelb/test/e2e/controller"
	_ "github.com/openelb/openelb/test/e2e/layer2"
	_ "github.com/openelb/openelb/test/e2e/vip"
)

func init() {
	klog.SetOutput(ginkgo.GinkgoWriter)

	// Register flags.
	config.CopyFlags(config.Flags, flag.CommandLine)
	framework.RegisterCommonFlags(flag.CommandLine)
	// framework.RegisterClusterFlags(flag.CommandLine)
}

func TestE2E(t *testing.T) {
	if framework.TestContext.KubeConfig == "" {
		framework.TestContext.KubeConfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	framework.AfterReadingAllFlags(&framework.TestContext)

	e2e.RunE2ETests(t)
}
