package nettool_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestNettool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nettool Suite")
}
