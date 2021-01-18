package leader

import "github.com/spf13/pflag"

type Options struct {
	LeaseDuration int
	RenewDeadline int
	RetryPeriod   int
}

func NewOptions() *Options {
	return &Options{
		LeaseDuration: 5,
		RenewDeadline: 3,
		RetryPeriod:   2,
	}
}

func (options *Options) AddFlags(fs *pflag.FlagSet) {
	fs.IntVar(&options.LeaseDuration, "lease-duration", options.LeaseDuration, "LeaseDuration is the duration that non-leader candidates will wait to force acquire leadership.")
	fs.IntVar(&options.RenewDeadline, "renew-deadline", options.RenewDeadline, "RenewDeadline is the duration that the acting master will retry refreshing leadership before giving up.")
	fs.IntVar(&options.RetryPeriod, "retry-period", options.RetryPeriod, "RetryPeriod is the duration the LeaderElector clients should wait between tries of actions.")
}
