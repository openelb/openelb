package log

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func InitLog(options *Options) {
	log := zap.New(zap.UseFlagOptions(&options.Options))
	ctrl.SetLogger(log)
}
