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
			Development:     true,
			Encoder:         nil,
			DestWriter:      nil,
			StacktraceLevel: zapcore.PanicLevel,
			TimeEncoder:     zapcore.RFC3339TimeEncoder,
			ZapOpts:         nil,
		}}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.BindFlags(flag.CommandLine)
}
