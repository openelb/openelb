package log

import (
	"flag"

	"github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Options struct {
	zap.Options
}

func NewOptions() *Options {
	return &Options{
		Options: zap.Options{
			Development:     false,
			Encoder:         nil,
			DestWritter:     nil,
			Level:           zapcore.InfoLevel,
			StacktraceLevel: zapcore.PanicLevel,
			ZapOpts:         nil,
		}}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.BindFlags(flag.CommandLine)
}
