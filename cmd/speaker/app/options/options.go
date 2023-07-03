package options

import (
	"github.com/openelb/openelb/pkg/log"
	"github.com/openelb/openelb/pkg/speaker/bgp"
	"github.com/openelb/openelb/pkg/speaker/vip"
	cliflag "k8s.io/component-base/cli/flag"
)

type OpenELBSpeakerOptions struct {
	MetricsAddr string
	Bgp         *bgp.BgpOptions
	Vip         *vip.VipOptions
	LogOptions  *log.Options
}

func NewOpenELBSpeakerOptions() *OpenELBSpeakerOptions {
	return &OpenELBSpeakerOptions{
		MetricsAddr: ":50053",
		Bgp:         bgp.NewBgpOptions(),
		Vip:         vip.NewVipOptions(),
		LogOptions:  log.NewOptions(),
	}
}

func (s *OpenELBSpeakerOptions) Validate() []error {
	var errs []error
	return errs
}

func (s *OpenELBSpeakerOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	s.Bgp.AddFlags(fss.FlagSet("bgp"))
	s.Vip.AddFlags(fss.FlagSet("vip"))
	s.LogOptions.AddFlags(fss.FlagSet("log"))

	fs := fss.FlagSet("generic")
	fs.StringVar(&s.MetricsAddr, "metrics-addr", s.MetricsAddr, "The address the metric endpoint binds to.")

	return fss
}
