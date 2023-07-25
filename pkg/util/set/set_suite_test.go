package set_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"testing"

	"github.com/onsi/ginkgo/v2/reporters"
)

func TestSet(t *testing.T) {
	log := zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter))
	ctrl.SetLogger(log)

	RegisterFailHandler(Fail)
	junitReporter := reporters.NewJUnitReporter("../../report/set_suite.xml")
	RunSpecsWithDefaultAndCustomReporters(t, "Set Suite", []Reporter{junitReporter})
}
