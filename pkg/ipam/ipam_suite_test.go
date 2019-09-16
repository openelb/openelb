package ipam_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestIpam(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ipam Suite")
}
