package options

import (
	"github.com/openelb/openelb/pkg/log"
	"github.com/openelb/openelb/pkg/manager"
	cliflag "k8s.io/component-base/cli/flag"
)

type OpenELBManagerOptions struct {
	*manager.GenericOptions
	LogOptions *log.Options
}

func NewOpenELBManagerOptions() *OpenELBManagerOptions {
	return &OpenELBManagerOptions{
		GenericOptions: manager.NewGenericOptions(),
		LogOptions:     log.NewOptions(),
	}
}

func (s *OpenELBManagerOptions) Validate() []error {
	var errs []error
	return errs
}

func (s *OpenELBManagerOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	s.GenericOptions.AddFlags(fss.FlagSet("generic"))
	s.LogOptions.AddFlags(fss.FlagSet("log"))

	return fss
}
