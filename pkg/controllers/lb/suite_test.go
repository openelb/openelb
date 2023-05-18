/*

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

package lb

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	networkv1alpha2 "github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/client"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/controllers/ipam"
	"github.com/openelb/openelb/pkg/manager"
	"github.com/openelb/openelb/pkg/util"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var testEnv *envtest.Environment
var stopCh context.Context
var cancel context.CancelFunc

var (
	node1 = &corev1.Node{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
		},
		Spec: corev1.NodeSpec{},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Type:    corev1.NodeInternalIP,
					Address: "192.168.0.1",
				},
			},
		},
	}

	node2 = &corev1.Node{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "node2",
		},
		Spec: corev1.NodeSpec{},
		Status: corev1.NodeStatus{
			Addresses: []corev1.NodeAddress{
				{
					Type:    corev1.NodeInternalIP,
					Address: "192.168.0.2",
				},
			},
		},
	}

	eip = &networkv1alpha2.Eip{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "testeip",
		},
		Spec: networkv1alpha2.EipSpec{
			Address: "10.0.0.1/24",
		},
		Status: networkv1alpha2.EipStatus{},
	}

	eipForCNI = &networkv1alpha2.Eip{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "testeip1",
			Labels: map[string]string{
				constant.OpenELBCNI: constant.OpenELBCNICalico,
			},
		},
		Spec: networkv1alpha2.EipSpec{
			Address: "10.0.1.1/24",
		},
		Status: networkv1alpha2.EipStatus{},
	}

	eipLayer2 = &networkv1alpha2.Eip{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "testeip2",
		},
		Spec: networkv1alpha2.EipSpec{
			Address:   "10.0.2.1/24",
			Protocol:  constant.OpenELBProtocolLayer2,
			Interface: "eth0",
		},
		Status: networkv1alpha2.EipStatus{},
	}

	svc = &corev1.Service{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testsvc",
			Namespace: "default",
			Annotations: map[string]string{
				constant.OpenELBAnnotationKey: constant.OpenELBAnnotationValue,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
		Status: corev1.ServiceStatus{},
	}
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	stopCh, cancel = context.WithCancel(context.Background())
	log := zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter))
	ctrl.SetLogger(log)

	RunSpecsWithDefaultAndCustomReporters(t,
		"LB Controller Suite",
		[]Reporter{})
}

var _ = BeforeSuite(func(done Done) {
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	// +kubebuilder:scaffold:scheme
	mgr, err := manager.NewManager(cfg, &manager.GenericOptions{
		WebhookPort:   443,
		MetricsAddr:   "0",
		ReadinessAddr: "0",
	})
	Expect(err).ToNot(HaveOccurred())
	Expect(mgr).ToNot(BeNil())

	// Setup all Controllers
	err = ipam.SetupIPAM(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = SetupServiceReconciler(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err := mgr.Start(stopCh)
		if err != nil {
			ctrl.Log.Error(err, "failed to start manager")
		}
	}()

	err = client.Client.Create(context.Background(), node1)
	Expect(err).ToNot(HaveOccurred())

	err = client.Client.Create(context.Background(), node2)
	Expect(err).ToNot(HaveOccurred())

	err = client.Client.Create(context.Background(), eip)
	Expect(err).ToNot(HaveOccurred())

	err = client.Client.Create(context.Background(), eipLayer2)
	Expect(err).ToNot(HaveOccurred())

	err = client.Client.Create(context.Background(), eipForCNI)
	Expect(err).ToNot(HaveOccurred())

	// wait other controller, like eip
	time.Sleep(1 * time.Second)

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func checkEipUsage(eip *networkv1alpha2.Eip, usage int) func() bool {
	return func() bool {
		clone := eip.DeepCopy()
		client.Client.Get(context.Background(), types.NamespacedName{
			Namespace: eip.Namespace,
			Name:      eip.Name,
		}, clone)

		if clone.Status.Usage == usage {
			return true
		}
		return false
	}
}

func checkSvc(svc *corev1.Service, fn func(dst *corev1.Service) bool) func() bool {
	return func() bool {
		clone := svc.DeepCopy()

		client.Client.Get(context.Background(), types.NamespacedName{
			Namespace: clone.Namespace,
			Name:      clone.Name,
		}, clone)

		if len(clone.Status.LoadBalancer.Ingress) <= 0 {
			return false
		}

		return fn(clone)
	}
}

var _ = Describe("OpenELB LoadBalancer Service", func() {
	BeforeEach(func() {
		Eventually(checkEipUsage(eip, 0), 3*time.Second).Should(BeTrue())
		Eventually(checkEipUsage(eipLayer2, 0), 3*time.Second).Should(BeTrue())

		clone := svc.DeepCopy()
		clone.Annotations[constant.OpenELBProtocolAnnotationKey] = eip.GetProtocol()
		clone.Annotations[constant.OpenELBEIPAnnotationKeyV1Alpha2] = eip.GetName()
		err := client.Client.Create(context.Background(), clone)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() error {
			return client.Client.Get(context.Background(), types.NamespacedName{
				Namespace: clone.Namespace,
				Name:      clone.Name,
			}, clone)
		}, 3*time.Second).ShouldNot(HaveOccurred())

		Eventually(checkEipUsage(eip, 1), 3*time.Second).Should(BeTrue())
		Eventually(checkEipUsage(eipLayer2, 0), 3*time.Second).Should(BeTrue())
	})

	AfterEach(func() {
		// Why does deleting a service delete the endpoint, but
		//creating a service does not create the endpoint.
		clone := svc.DeepCopy()
		err := client.Client.Delete(context.Background(), clone)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			err = client.Client.Get(context.Background(), types.NamespacedName{
				Namespace: clone.Namespace,
				Name:      clone.Name,
			}, clone)
			return k8serrors.IsNotFound(err)
		}, 3*time.Second).Should(Equal(true))

		Eventually(checkEipUsage(eip, 0), 3*time.Second).Should(BeTrue())
		Eventually(checkEipUsage(eipLayer2, 0), 3*time.Second).Should(BeTrue())
	})

	It("Should add FinalizerName", func() {
		Eventually(checkSvc(svc, func(dst *corev1.Service) bool {
			return util.ContainsString(dst.Finalizers, constant.FinalizerName)
		}), 3*time.Second).Should(Equal(true))
	})

	It("LoadBalancer Service should assign ingress ip", func() {
		Eventually(checkSvc(svc, func(dst *corev1.Service) bool {
			if len(dst.Status.LoadBalancer.Ingress) == 0 {
				return false
			}
			return eip.Contains(net.ParseIP(dst.Status.LoadBalancer.Ingress[0].IP))
		}), 3*time.Second).Should(Equal(true))
	})

})

func updateSvc(origin *corev1.Service, fn func(dst *corev1.Service)) {
	clone := origin.DeepCopy()

	retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		client.Client.Get(context.Background(), types.NamespacedName{
			Namespace: clone.Namespace,
			Name:      clone.Name,
		}, clone)
		fn(clone)
		return client.Client.Update(context.Background(), clone)
	})
}

func updateEndpoints(origin *corev1.Endpoints, fn func(dst *corev1.Endpoints)) {
	clone := origin.DeepCopy()

	retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		err := client.Client.Get(context.Background(), types.NamespacedName{
			Namespace: clone.Namespace,
			Name:      clone.Name,
		}, clone)
		if err != nil {
			return err
		}
		fn(clone)
		return client.Client.Update(context.Background(), clone)
	})
}
