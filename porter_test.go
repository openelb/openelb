package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/kubesphere/porter"
)

var _ = Describe("Porter", func() {
	It("Should ok", func() {
		Expect(A()).ShouldNot(BeEmpty())
	})
})
