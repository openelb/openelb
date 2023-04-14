package options

import "github.com/spf13/pflag"

type Options struct {
	Port int
}

func NewOptions() *Options {
	return &Options{
		Port: 8080,
	}
}

func (options *Options) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&options.Port, "http-port", options.Port, "The port that the http server serves at")
}
