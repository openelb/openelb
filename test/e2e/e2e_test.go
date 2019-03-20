package e2e_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	devopsv1alpha1 "github.com/kubesphere/porter/pkg/apis/network/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
)

var _ = Describe("E2e", func() {
	It("Should work well when using samples", func() {
		eip := &devopsv1alpha1.EIP{}
		reader, err := os.Open(workspace + "/config/samples/network_v1alpha1_eip.yaml")
		Expect(err).NotTo(HaveOccurred(), "Cannot read sample yamls")
		err = yaml.NewYAMLOrJSONDecoder(reader, 10).Decode(eip)
		Expect(err).NotTo(HaveOccurred(), "Cannot unmarshal yamls")
		err = testClient.Create(context.TODO(), eip)
		Expect(err).NotTo(HaveOccurred())
		defer testClient.Delete(context.TODO(), eip)

		//apply service
		cmd := exec.Command("kubectl", "apply", "-f", workspace+"/config/samples/service.yaml")
		Expect(cmd.Run()).ShouldNot(HaveOccurred())
		defer func() {
			cmd := exec.Command("kubectl", "delete", "-f", workspace+"/config/samples/service.yaml")
			Expect(cmd.Run()).ShouldNot(HaveOccurred())
		}()
		//Service should get its eip
		Eventually(func() error {
			service := &corev1.Service{}
			err := testClient.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "mylbapp"}, service)
			if err != nil {
				return err
			}
			if service.Spec.ExternalIPs[0] == eip.Spec.Address && service.Status.LoadBalancer.Ingress[0].IP == eip.Spec.Address {
				return nil
			}
			return fmt.Errorf("Failed")
		}, time.Second, 2*time.Minute).Should(Succeed())

		//check route in bird

	})
	//install eip
})
