package util_test

import (
	"github.com/kubesphere/porterlb/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {
	Context("Tests of strings function", func() {
		It("Should work well", func() {
			test := []string{"a", "b", "cc", "ddd"}
			Expect(util.ContainsString(test, "a")).To(BeTrue())
			Expect(util.ContainsString(test, "aaa")).To(BeFalse())
			remove := util.RemoveString(test, "a")
			Expect(util.ContainsString(remove, "a")).To(BeFalse())
			Expect(util.ContainsString(remove, "ddd")).To(BeTrue())
		})
	})
})
