package framework

import (
	"context"

	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WaitDaemonsetPresentFitWith wait daemonset present on cluster sync with fit func.
func WaitDaemonsetPresentFitWith(client client.Client, namespace, name string, fit func(DaemonSetCondition *appsv1.DaemonSet) bool) {
	gomega.Expect(client).ShouldNot(gomega.BeNil())

	klog.Infof("Waiting for daemonset(%s/%s) synced", namespace, name)
	gomega.Eventually(func() bool {
		ds := &appsv1.DaemonSet{}
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, ds)
		if err != nil {
			return false
		}
		return fit(ds)
	}, pollTimeout, pollInterval).Should(gomega.Equal(true))
}
