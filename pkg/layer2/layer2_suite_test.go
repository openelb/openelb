package layer2_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestLayer2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Layer2 Suite")
}
