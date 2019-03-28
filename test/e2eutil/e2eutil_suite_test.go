package e2eutil_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestE2eutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2eutil Suite")
}
