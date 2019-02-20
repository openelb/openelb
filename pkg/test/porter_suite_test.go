package test

import (
	"testing"

	bgpserver "github.com/kubesphere/porter/pkg/bgp/serverd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUtil(t *testing.T) {
	RegisterFailHandler(Fail)
	ready := make(chan interface{})
	go bgpserver.RunAlone(ready)
	<-ready
	RunSpecs(t, "Util suite")
}
