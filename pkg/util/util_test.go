package util

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/openelb/openelb/pkg/constant"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Util", func() {
	Context("Tests of strings function", func() {
		It("Should work well", func() {
			test := []string{"a", "b", "cc", "ddd"}
			Expect(ContainsString(test, "a")).To(BeTrue())
			Expect(ContainsString(test, "aaa")).To(BeFalse())
			remove := RemoveString(test, "a")
			Expect(ContainsString(remove, "a")).To(BeFalse())
			Expect(ContainsString(remove, "ddd")).To(BeTrue())
		})
	})

	It("DutyOfCNI should work well", func() {
		Expect(DutyOfCNI(nil, &metav1.ObjectMeta{
			Labels: map[string]string{
				constant.OpenELBCNI: constant.OpenELBCNICalico,
			},
		})).To(BeTrue())
		Expect(DutyOfCNI(nil, &metav1.ObjectMeta{
			Labels: map[string]string{
				"test": "test",
			},
		})).To(BeFalse())
		Expect(DutyOfCNI(&metav1.ObjectMeta{
			Labels: map[string]string{
				"test": "test",
			},
		}, &metav1.ObjectMeta{
			Labels: map[string]string{
				constant.OpenELBCNI: constant.OpenELBCNICalico,
			},
		})).To(BeFalse())
		Expect(DutyOfCNI(&metav1.ObjectMeta{
			Labels: map[string]string{
				constant.OpenELBCNI: constant.OpenELBCNICalico,
			},
		}, &metav1.ObjectMeta{
			Labels: map[string]string{
				constant.OpenELBCNI: constant.OpenELBCNICalico,
			},
		})).To(BeTrue())
		Expect(DutyOfCNI(&metav1.ObjectMeta{
			Labels: map[string]string{
				"test": "test",
			},
		}, &metav1.ObjectMeta{
			Labels: map[string]string{
				"test": "test",
			},
		})).To(BeFalse())
	})
})
