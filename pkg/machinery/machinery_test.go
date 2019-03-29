package machinery_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/kubesphere/porter/pkg/machinery"
)

var _ = Describe("Machinery", func() {
	Context("Testing Requeue Machinery", func() {
		It("Should caculate correct requeue time", func() {
			test1 := &corev1.Service{}
			test1.Namespace = "default"
			test1.Name = "test1"
			Expect(GetRequeueTime(test1)).To(Equal(RequeueBaseTime * time.Second))
			Expect(GetRequeueTime(test1)).To(Equal((RequeueBaseTime + RequeueStep) * time.Second))

			test2 := test1.DeepCopy()
			test2.Name = "test2"
			Expect(GetRequeueTime(test2)).To(Equal(RequeueBaseTime * time.Second))
			for index := 0; index < 20; index++ {
				Expect(GetRequeueTime(test2)).ShouldNot(BeNumerically(">", RequeueMaximum*time.Second))
			}
		})
	})
})
