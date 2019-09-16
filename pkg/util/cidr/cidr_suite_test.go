package cidr

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestCidr(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cidr Suite")
}
