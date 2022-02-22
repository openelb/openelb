package options

import (
	"github.com/openelb/openelb/pkg/leader-elector"
	"github.com/openelb/openelb/pkg/log"
	"github.com/openelb/openelb/pkg/manager"
	"github.com/openelb/openelb/pkg/speaker/bgp"
	cliflag "k8s.io/component-base/cli/flag"
)

type OpenELBManagerOptions struct {
	Bgp *bgp.BgpOptions
	*manager.GenericOptions
	LogOptions *log.Options
	Leader     *leader.Options
}

func NewOpenELBManagerOptions() *OpenELBManagerOptions {
	return &OpenELBManagerOptions{
		Bgp:            bgp.NewBgpOptions(),
		GenericOptions: manager.NewGenericOptions(),
		LogOptions:     log.NewOptions(),
		Leader:         leader.NewOptions(),
	}
}

func (s *OpenELBManagerOptions) Validate() []error {
	var errs []error
	return errs
}

func (s *OpenELBManagerOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	s.Bgp.AddFlags(fss.FlagSet("bgp"))
	s.GenericOptions.AddFlags(fss.FlagSet("generic"))
	s.LogOptions.AddFlags(fss.FlagSet("log"))
	s.Leader.AddFlags(fss.FlagSet("leader"))

	return fss
}
