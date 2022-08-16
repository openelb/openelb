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

package bgp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/manager"
	"github.com/openelb/openelb/pkg/manager/client"
	"github.com/openelb/openelb/pkg/speaker/bgp"
	"github.com/openelb/openelb/pkg/util"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
var bgpServer *bgp.Bgp

var (
	node1 = &corev1.Node{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
			Labels: map[string]string{
				"kubernetes.io/hostname": "node1",
			},
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
			Labels: map[string]string{
				"kubernetes.io/hostname": "node2",
			},
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

	bgpConf = &v1alpha2.BgpConf{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Spec: v1alpha2.BgpConfSpec{
			As:         65001,
			RouterId:   "192.168.1.1",
			ListenPort: 17901,
		},
		Status: v1alpha2.BgpConfStatus{},
	}

	bgpPeer = &v1alpha2.BgpPeer{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "peer1",
		},
		Spec: v1alpha2.BgpPeerSpec{
			Conf: &v1alpha2.PeerConf{
				NeighborAddress: "192.168.1.2",
				PeerAs:          65001,
			},
		},
		Status: v1alpha2.BgpPeerStatus{},
	}
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	stopCh = make(chan struct{})
	log := zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter))
	ctrl.SetLogger(log)

	RunSpecsWithDefaultAndCustomReporters(t,
		"BGP Controller Suite",
		[]Reporter{printer.NewlineReporter{}})

}

var _ = BeforeSuite(func(done Done) {
	syncStatusPeriod = 3

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
	bgpServer = bgp.NewGoBgpd(bgp.NewBgpOptions())
	bgpServer.Start(stopCh)
	err = SetupBgpPeerReconciler(bgpServer, mgr)
	Expect(err).ToNot(HaveOccurred())
	err = SetupBgpConfReconciler(bgpServer, mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err := mgr.Start(stopCh)
		if err != nil {
			ctrl.Log.Error(err, "failed to start manager")
		}
	}()

	err = client.Client.Create(context.Background(), node1)
	Expect(err).ToNot(HaveOccurred())
	os.Setenv(constant.EnvNodeName, node1.Name)

	err = client.Client.Create(context.Background(), node2)
	Expect(err).ToNot(HaveOccurred())

	SetDefaultEventuallyTimeout(3 * time.Second)

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	close(stopCh)
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

var _ = Describe("Test GoBGP Controller", func() {
	When("BgpConf has label "+constant.OpenELBCNI, func() {
		BeforeEach(func() {
			clone := bgpConf.DeepCopy()

			err := util.Create(context.Background(), client.Client, clone, func() error {
				if clone.Labels == nil {
					clone.Labels = make(map[string]string)
				}
				clone.Labels[constant.OpenELBCNI] = constant.OpenELBCNICalico
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			clone := bgpConf.DeepCopy()
			err := client.Client.Delete(context.Background(), clone)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				err = client.Client.Get(context.Background(), types.NamespacedName{
					Namespace: clone.Namespace,
					Name:      clone.Name,
				}, clone)
				return k8serrors.IsNotFound(err)
			}, 3*time.Second).Should(Equal(true))
		})

		It("BgpConf should not have Finalizer", func() {
			Eventually(checkBgpConf(bgpConf, func(dst *v1alpha2.BgpConf) bool {
				return util.ContainsString(dst.Finalizers, constant.FinalizerName)
			}), 3*time.Second).Should(Equal(false))
		})
	})

	When("BgpConf has no label "+constant.OpenELBCNI, func() {
		BeforeEach(func() {
			clone := bgpConf.DeepCopy()
			err := util.Create(context.Background(), client.Client, clone, func() error {
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			clone := bgpConf.DeepCopy()
			err := client.Client.Delete(context.Background(), clone)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				err = client.Client.Get(context.Background(), types.NamespacedName{
					Namespace: clone.Namespace,
					Name:      clone.Name,
				}, clone)
				return k8serrors.IsNotFound(err)
			}, 3*time.Second).Should(Equal(true))
		})

		It("BgpConf should have Finalizer", func() {
			clone := bgpConf.DeepCopy()
			Eventually(util.Check(context.Background(), client.Client, clone, func() bool {
				return util.ContainsString(clone.Finalizers, constant.FinalizerName)
			}), 3*time.Second).Should(Equal(false))
		})

		Context("BgpConf status should be updated", func() {
			It("routeId be set", func() {
				clone := bgpConf.DeepCopy()
				Eventually(util.Check(context.Background(), client.Client, clone, func() bool {
					if clone.Status.NodesConfStatus == nil {
						return false
					}

					if clone.Status.NodesConfStatus[util.GetNodeName()].RouterId == bgpConf.Spec.RouterId {
						return true
					}

					return false
				}), 35*time.Second).Should(Equal(false))
			})

			It("routerId is empty", func() {
				updateBgpConf(bgpConf, func(dst *v1alpha2.BgpConf) {
					dst.Spec.RouterId = ""
				})

				Eventually(checkBgpConf(bgpConf, func(dst *v1alpha2.BgpConf) bool {
					if dst.Status.NodesConfStatus == nil {
						return false
					}

					if dst.Status.NodesConfStatus[util.GetNodeName()].RouterId == node1.Status.Addresses[0].Address {
						return true
					}

					return false
				}), 35*time.Second).Should(Equal(true))
			})
		})

		Context("has bgpPeer", func() {
			When("bgpPeer has no cni annotation", func() {
				BeforeEach(func() {
					clone := bgpPeer.DeepCopy()
					err := client.Client.Create(context.Background(), clone)
					Expect(err).ToNot(HaveOccurred())
					Eventually(func() error {
						return client.Client.Get(context.Background(), types.NamespacedName{
							Namespace: clone.Namespace,
							Name:      clone.Name,
						}, clone)
					}, 3*time.Second).ShouldNot(HaveOccurred())
				})

				AfterEach(func() {
					clone := bgpPeer.DeepCopy()
					err := client.Client.Delete(context.Background(), clone)
					Expect(err).ToNot(HaveOccurred())
					Eventually(func() bool {
						err = client.Client.Get(context.Background(), types.NamespacedName{
							Namespace: clone.Namespace,
							Name:      clone.Name,
						}, clone)
						return k8serrors.IsNotFound(err)
					}, 3*time.Second).Should(Equal(true))
				})

				It("BgpPeer should have Finalizer", func() {
					Eventually(checkBgpPeer(bgpPeer, func(dst *v1alpha2.BgpPeer) bool {
						return util.ContainsString(dst.Finalizers, constant.FinalizerName)
					}), 3*time.Second).Should(Equal(true))
				})

				It("BgpPeer should have status", func() {
					Eventually(checkBgpPeer(bgpPeer, func(dst *v1alpha2.BgpPeer) bool {
						if dst.Status.NodesPeerStatus == nil {
							return false
						}

						if _, ok := dst.Status.NodesPeerStatus[util.GetNodeName()]; ok {
							return true
						}

						return false
					}), 35*time.Second).Should(Equal(true))
				})

				It("BgpPeer should delete status when it cannot match node ", func() {
					updateBgpPeer(bgpPeer, func(dst *v1alpha2.BgpPeer) {
						dst.Spec.NodeSelector = &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kubernetes.io/hostname": node2.Name,
							},
							MatchExpressions: nil,
						}
					})

					Eventually(checkBgpPeer(bgpPeer, func(dst *v1alpha2.BgpPeer) bool {
						if dst.Status.NodesPeerStatus == nil {
							return true
						}

						return false
					}), 35*time.Second).Should(Equal(true))
				})

				It("BgpPeer should have status when it match node ", func() {
					updateBgpPeer(bgpPeer, func(dst *v1alpha2.BgpPeer) {
						dst.Spec.NodeSelector = &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kubernetes.io/hostname": node1.Name,
							},
							MatchExpressions: nil,
						}
					})

					Eventually(checkBgpPeer(bgpPeer, func(dst *v1alpha2.BgpPeer) bool {
						if dst.Status.NodesPeerStatus == nil {
							return false
						}

						if _, ok := dst.Status.NodesPeerStatus[util.GetNodeName()]; ok {
							return true
						}

						return false
					}), 35*time.Second).Should(Equal(true))
				})

				It("BgpPeer should have status when modify bgpconf ", func() {
					updateBgpConf(bgpConf, func(dst *v1alpha2.BgpConf) {
						dst.Spec.RouterId = ""
					})

					Eventually(checkBgpPeer(bgpPeer, func(dst *v1alpha2.BgpPeer) bool {
						if dst.Status.NodesPeerStatus == nil {
							return false
						}

						if _, ok := dst.Status.NodesPeerStatus[util.GetNodeName()]; ok {
							return true
						}

						return false
					}), 35*time.Second).Should(Equal(true))
				})

				It("BgpPeer should delete status when delete bgpconf ", func() {
					cloneBgpConf := bgpConf.DeepCopy()
					err := client.Client.Delete(context.Background(), cloneBgpConf)
					Expect(err).ToNot(HaveOccurred())
					Eventually(func() bool {
						err = client.Client.Get(context.Background(), types.NamespacedName{
							Namespace: cloneBgpConf.Namespace,
							Name:      cloneBgpConf.Name,
						}, cloneBgpConf)
						return k8serrors.IsNotFound(err)
					}, 3*time.Second).Should(Equal(true))

					Eventually(checkBgpPeer(bgpPeer, func(dst *v1alpha2.BgpPeer) bool {
						if dst.Status.NodesPeerStatus == nil {
							return true
						}

						return false
					}), 35*time.Second).Should(Equal(true))

					cloneBgpConf = bgpConf.DeepCopy()
					err = client.Client.Create(context.Background(), cloneBgpConf)
					Expect(err).ToNot(HaveOccurred())
					Eventually(func() error {
						return client.Client.Get(context.Background(), types.NamespacedName{
							Namespace: cloneBgpConf.Namespace,
							Name:      cloneBgpConf.Name,
						}, cloneBgpConf)
					}, 3*time.Second).ShouldNot(HaveOccurred())

					Eventually(checkBgpPeer(bgpPeer, func(dst *v1alpha2.BgpPeer) bool {
						if dst.Status.NodesPeerStatus == nil {
							return false
						}

						if _, ok := dst.Status.NodesPeerStatus[util.GetNodeName()]; ok {
							return true
						}

						return false
					}), 35*time.Second).Should(Equal(true))
				})
			})

			When("bgpPeer has cni annotation", func() {
				BeforeEach(func() {
					clone := bgpPeer.DeepCopy()
					err := util.Create(context.Background(), client.Client, clone, func() error {
						if clone.Labels == nil {
							clone.Labels = make(map[string]string)
						}
						clone.Labels[constant.OpenELBCNI] = constant.OpenELBCNICalico
						return nil
					})
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					clone := bgpPeer.DeepCopy()
					err := client.Client.Delete(context.Background(), clone)
					Expect(err).ToNot(HaveOccurred())
					Eventually(func() bool {
						err = client.Client.Get(context.Background(), types.NamespacedName{
							Namespace: clone.Namespace,
							Name:      clone.Name,
						}, clone)
						return k8serrors.IsNotFound(err)
					}, 3*time.Second).Should(Equal(true))
				})
				It("BgpPeer should not have Finalizer", func() {
					Eventually(checkBgpPeer(bgpPeer, func(dst *v1alpha2.BgpPeer) bool {
						return util.ContainsString(dst.Finalizers, constant.FinalizerName)
					}), 3*time.Second).Should(Equal(false))
				})
			})
		})
	})
})

func checkBgpConf(bgpConf *v1alpha2.BgpConf, fn func(dst *v1alpha2.BgpConf) bool) func() bool {
	return func() bool {
		clone := bgpConf.DeepCopy()

		client.Client.Get(context.Background(), types.NamespacedName{
			Namespace: clone.Namespace,
			Name:      clone.Name,
		}, clone)

		return fn(clone)
	}
}

func checkBgpPeer(bgpPeer *v1alpha2.BgpPeer, fn func(dst *v1alpha2.BgpPeer) bool) func() bool {
	return func() bool {
		clone := bgpPeer.DeepCopy()

		client.Client.Get(context.Background(), types.NamespacedName{
			Namespace: clone.Namespace,
			Name:      clone.Name,
		}, clone)

		return fn(clone)
	}
}

func updateBgpConf(origin *v1alpha2.BgpConf, fn func(dst *v1alpha2.BgpConf)) {
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

func updateBgpPeer(origin *v1alpha2.BgpPeer, fn func(dst *v1alpha2.BgpPeer)) {
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
