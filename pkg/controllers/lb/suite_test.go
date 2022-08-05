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
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	networkv1alpha2 "github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/controllers/ipam"
	"github.com/openelb/openelb/pkg/manager"
	"github.com/openelb/openelb/pkg/manager/client"
	"github.com/openelb/openelb/pkg/speaker"
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
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var testEnv *envtest.Environment
var stopCh chan struct{}

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

	endpoints = &corev1.Endpoints{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP:       "192.168.0.1",
						NodeName: &node1.Name,
					},
				},
				NotReadyAddresses: nil,
				Ports: []corev1.EndpointPort{
					{
						Port:     80,
						Protocol: corev1.ProtocolTCP,
					},
				},
			},
		},
	}

	bgpFakeSpeak    = speaker.NewFake()
	layer2FakeSpeak = speaker.NewFake()
	dummySpeak      = speaker.NewFake()
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	stopCh = make(chan struct{})
	log := zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter))
	ctrl.SetLogger(log)

	RunSpecsWithDefaultAndCustomReporters(t,
		"LB Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
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

	err = speaker.RegisterSpeaker(eip.GetSpeakerName(), bgpFakeSpeak)
	Expect(err).ToNot(HaveOccurred())
	err = client.Client.Create(context.Background(), eip)
	Expect(err).ToNot(HaveOccurred())

	err = speaker.RegisterSpeaker(eipLayer2.GetSpeakerName(), layer2FakeSpeak)
	Expect(err).ToNot(HaveOccurred())
	err = client.Client.Create(context.Background(), eipLayer2)
	Expect(err).ToNot(HaveOccurred())

	err = speaker.RegisterSpeaker(eipForCNI.GetSpeakerName(), dummySpeak)
	Expect(err).ToNot(HaveOccurred())
	err = client.Client.Create(context.Background(), eipForCNI)
	Expect(err).ToNot(HaveOccurred())

	// wait other controller, like eip
	time.Sleep(1 * time.Second)

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	close(stopCh)
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

		cloneEP := endpoints.DeepCopy()
		err := client.Client.Create(context.Background(), cloneEP)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() error {
			return client.Client.Get(context.Background(), types.NamespacedName{
				Namespace: cloneEP.Namespace,
				Name:      cloneEP.Name,
			}, cloneEP)
		}, 3*time.Second).ShouldNot(HaveOccurred())

		clone := svc.DeepCopy()
		err = client.Client.Create(context.Background(), clone)
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

		cloneEP := endpoints.DeepCopy()
		Eventually(func() bool {
			err := client.Client.Get(context.Background(), types.NamespacedName{
				Namespace: cloneEP.Namespace,
				Name:      cloneEP.Name,
			}, cloneEP)
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
			return bgpFakeSpeak.Equal(dst.Status.LoadBalancer.Ingress[0].IP,
				[]string{
					node2.Name,
					node1.Name,
				})
		}), 3*time.Second).Should(Equal(true))
	})

	It("Nexthops should be all nodes", func() {
		Eventually(checkSvc(svc, func(dst *corev1.Service) bool {
			return bgpFakeSpeak.Equal(dst.Status.LoadBalancer.Ingress[0].IP,
				[]string{
					node2.Name,
					node1.Name,
				})
		}), 3*time.Second).Should(Equal(true))
	})

	When("Endpoint is empty", func() {
		BeforeEach(func() {
			updateEndpoints(endpoints, func(dst *corev1.Endpoints) {
				dst.Subsets = []corev1.EndpointSubset{
					{
						NotReadyAddresses: []corev1.EndpointAddress{
							{
								IP:       node1.Status.Addresses[0].Address,
								NodeName: &node1.Name,
							},
						},
						Ports: []corev1.EndpointPort{
							{
								Port:     80,
								Protocol: corev1.ProtocolTCP,
							},
						},
					},
				}
			})
		})
		It("the nexthops should not be empty", func() {
			Eventually(checkSvc(svc, func(dst *corev1.Service) bool {
				return bgpFakeSpeak.Equal(dst.Status.LoadBalancer.Ingress[0].IP,
					[]string{
						node2.Name,
						node1.Name,
					})
			}), 3*time.Second).Should(Equal(true))
		})
	})

	Context("ExternalTrafficPolicy == ServiceExternalTrafficPolicyTypeLocal", func() {
		BeforeEach(func() {
			updateSvc(svc, func(dst *corev1.Service) {
				dst.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeLocal
			})
		})

		It("external local service should forward to loacl node", func() {
			Eventually(checkSvc(svc, func(dst *corev1.Service) bool {
				return bgpFakeSpeak.Equal(dst.Status.LoadBalancer.Ingress[0].IP,
					[]string{
						node1.Name,
					})
			}), 3*time.Second).Should(Equal(true))
		})

		It("nexthop should change when endpoint changed", func() {
			updateEndpoints(endpoints, func(dst *corev1.Endpoints) {
				dst.Subsets = []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{
								IP:       node2.Status.Addresses[0].Address,
								NodeName: &node2.Name,
							},
						},
						NotReadyAddresses: []corev1.EndpointAddress{
							{
								IP:       node1.Status.Addresses[0].Address,
								NodeName: &node1.Name,
							},
						},
						Ports: []corev1.EndpointPort{
							{
								Port:     80,
								Protocol: corev1.ProtocolTCP,
							},
						},
					},
				}
			})

			Eventually(checkSvc(svc, func(dst *corev1.Service) bool {
				return bgpFakeSpeak.Equal(dst.Status.LoadBalancer.Ingress[0].IP,
					[]string{
						node2.Name,
					})
			}), 3*time.Second).Should(Equal(true))
		})
	})

	When("Eip has label "+constant.OpenELBCNI, func() {
		BeforeEach(func() {
			updateSvc(svc, func(dst *corev1.Service) {
				dst.Labels[constant.OpenELBCNI] = constant.OpenELBCNICalico
			})
		})

		It("Speaker should be dummy", func() {
			Eventually(checkSvc(svc, func(dst *corev1.Service) bool {
				return !dummySpeak.Equal(dst.Status.LoadBalancer.Ingress[0].IP,
					[]string{
						node2.Name,
						node1.Name,
					})
			}), 3*time.Second).Should(Equal(true))
		})
	})

	Context("Change to Layer2 LoadBalancer Service", func() {
		BeforeEach(func() {
			updateSvc(svc, func(dst *corev1.Service) {
				dst.Annotations[constant.OpenELBProtocolAnnotationKey] = constant.OpenELBProtocolLayer2
			})
		})

		It("Nexthops should not be all nodes", func() {
			Eventually(checkSvc(svc, func(dst *corev1.Service) bool {
				return !layer2FakeSpeak.Equal(dst.Status.LoadBalancer.Ingress[0].IP,
					[]string{
						node2.Name,
						node1.Name,
					})
			}), 3*time.Second).Should(Equal(true))
		})

		Context("ExternalTrafficPolicy == ServiceExternalTrafficPolicyTypeLocal", func() {
			BeforeEach(func() {
				updateSvc(svc, func(dst *corev1.Service) {
					dst.Spec.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeLocal
				})
			})

			It("layer2 service should have annotation", func() {
				Eventually(checkEipUsage(eip, 0), 3*time.Second).Should(BeTrue())
				Eventually(checkEipUsage(eipLayer2, 1), 3*time.Second).Should(BeTrue())

				Eventually(checkSvc(svc, func(dst *corev1.Service) bool {
					if dst.Annotations[constant.OpenELBLayer2Annotation] == node1.Name {
						return true
					}

					return layer2FakeSpeak.Equal(dst.Status.LoadBalancer.Ingress[0].IP, []string{
						node1.Name,
					})
				}), 3*time.Second).Should(Equal(true))

				By("When the endpoint changes, the annotation changes at the same time.")
				updateEndpoints(endpoints, func(dst *corev1.Endpoints) {
					dst.Subsets = []corev1.EndpointSubset{
						{
							Addresses: []corev1.EndpointAddress{
								{
									IP:       node2.Status.Addresses[0].Address,
									NodeName: &node2.Name,
								},
							},
							NotReadyAddresses: []corev1.EndpointAddress{
								{
									IP:       node1.Status.Addresses[0].Address,
									NodeName: &node1.Name,
								},
							},
							Ports: []corev1.EndpointPort{
								{
									Port:     80,
									Protocol: corev1.ProtocolTCP,
								},
							},
						},
					}
				})

				Eventually(checkSvc(svc, func(dst *corev1.Service) bool {
					if dst.Annotations[constant.OpenELBLayer2Annotation] == node2.Name {
						return true
					}

					return layer2FakeSpeak.Equal(dst.Status.LoadBalancer.Ingress[0].IP, []string{
						node2.Name,
					})
				}), 3*time.Second).Should(Equal(true))
			})
		})
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
