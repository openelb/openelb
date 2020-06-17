package options

import (
	"github.com/spf13/pflag"
)

type GenericOptions struct {
	MetricsAddr          string
	ReadinessAddr        string
	EnableLeaderElection bool
}

func NewGenericOptions() *GenericOptions {
	return &GenericOptions{
		MetricsAddr:          ":8080",
		ReadinessAddr:        ":8000",
		EnableLeaderElection: true,
	}
}

func (options *GenericOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&options.MetricsAddr, "metrics-addr", options.MetricsAddr, "The address the metric endpoint binds to.")
	fs.BoolVar(&options.EnableLeaderElection, "enable-leader-election", options.EnableLeaderElection, "Whether to enable leader "+
		"election. This field should be enabled when porter manager deployed with multiple replicas.")
	fs.StringVar(&options.ReadinessAddr, "readiness-addr", options.ReadinessAddr, "The address readinessProbe used")
}
