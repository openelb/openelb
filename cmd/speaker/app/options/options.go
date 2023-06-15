package options

import (
	"github.com/openelb/openelb/pkg/log"
	"github.com/openelb/openelb/pkg/speaker/bgp"
	cliflag "k8s.io/component-base/cli/flag"
)

type OpenELBSpeakerOptions struct {
	Bgp        *bgp.BgpOptions
	LogOptions *log.Options
}

func NewOpenELBSpeakerOptions() *OpenELBSpeakerOptions {
	return &OpenELBSpeakerOptions{
		Bgp:        bgp.NewBgpOptions(),
		LogOptions: log.NewOptions(),
	}
}

func (s *OpenELBSpeakerOptions) Validate() []error {
	var errs []error
	return errs
}

func (s *OpenELBSpeakerOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	s.Bgp.AddFlags(fss.FlagSet("bgp"))
	s.LogOptions.AddFlags(fss.FlagSet("log"))

	return fss
}
