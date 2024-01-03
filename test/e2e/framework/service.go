package framework

import (
	"context"

	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WaitServicePresentFitWith wait service present on cluster sync with fit func.
func WaitServicePresentFitWith(client client.Client, namespace, name string, fit func(service *corev1.Service) bool) {
	gomega.Expect(client).ShouldNot(gomega.BeNil())

	klog.Infof("Waiting for service(%s/%s) synced", namespace, name)
	gomega.Eventually(func() bool {
		svc := &corev1.Service{}
		err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, svc)
		if err != nil {
			return false
		}
		return fit(svc)
	}, pollTimeout, pollInterval).Should(gomega.Equal(true))
}
