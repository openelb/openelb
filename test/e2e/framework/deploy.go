package framework

import (
	"context"

	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WaitDeploymentPresentFitWith wait deployment present on cluster sync with fit func.
func WaitDeploymentPresentFitWith(client client.Client, namespace, name string, fit func(deployment *appsv1.Deployment) bool) {
	gomega.Expect(client).ShouldNot(gomega.BeNil())

	klog.Infof("Waiting for deployment(%s/%s) synced", namespace, name)
	gomega.Eventually(func() bool {
		deploy := &appsv1.Deployment{}
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, deploy)
		if err != nil {
			return false
		}
		return fit(deploy)
	}, pollTimeout, pollInterval).Should(gomega.Equal(true))
}
