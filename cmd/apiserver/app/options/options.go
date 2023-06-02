package options

import (
	"github.com/openelb/openelb/pkg/log"
	server "github.com/openelb/openelb/pkg/server/options"
	cliflag "k8s.io/component-base/cli/flag"
)

type OpenELBApiServerOptions struct {
	HTTPOptions *server.Options
	LogOptions  *log.Options
}

func NewOpenELBApiServerOptions() *OpenELBApiServerOptions {
	return &OpenELBApiServerOptions{
		HTTPOptions: server.NewOptions(),
		LogOptions:  log.NewOptions(),
	}
}

func (s *OpenELBApiServerOptions) Validate() []error {
	var errs []error
	return errs
}

func (s *OpenELBApiServerOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	s.HTTPOptions.AddFlags(fss.FlagSet("http"))
	s.LogOptions.AddFlags(fss.FlagSet("log"))

	return fss
}
