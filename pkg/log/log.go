package log

import (
	"github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type LogOptions struct {
	*zap.Options
}

func NewLogOptions() *LogOptions {
	return &LogOptions{
		Options: &zap.Options{
			Development: false,
		},
	}
}

func (options *LogOptions) AddFlags(fs *pflag.FlagSet) {
	// Set Development mode value
	fs.BoolVar(&options.Development, "zap-devel", options.Development,
		"Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). "+
			"Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)")
}

func InitLog(options *LogOptions) {
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(options.Options)))
}
