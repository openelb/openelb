package options

import (
	"github.com/openelb/openelb/pkg/leader-elector"
	"github.com/openelb/openelb/pkg/log"
	"github.com/openelb/openelb/pkg/manager"
	server "github.com/openelb/openelb/pkg/server/options"
	"github.com/openelb/openelb/pkg/speaker/bgp"
	cliflag "k8s.io/component-base/cli/flag"
)

type OpenELBSpeakerOptions struct {
	Bgp *bgp.BgpOptions
	*manager.GenericOptions
	LogOptions  *log.Options
	Leader      *leader.Options
	HTTPOptions *server.Options
}

func NewOpenELBSpeakerOptions() *OpenELBSpeakerOptions {
	return &OpenELBSpeakerOptions{
		Bgp:            bgp.NewBgpOptions(),
		GenericOptions: manager.NewGenericOptions(),
		LogOptions:     log.NewOptions(),
		Leader:         leader.NewOptions(),
		HTTPOptions:    server.NewOptions(),
	}
}

func (s *OpenELBSpeakerOptions) Validate() []error {
	var errs []error
	return errs
}

func (s *OpenELBSpeakerOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	s.Bgp.AddFlags(fss.FlagSet("bgp"))
	s.GenericOptions.AddFlags(fss.FlagSet("generic"))
	s.LogOptions.AddFlags(fss.FlagSet("log"))
	s.Leader.AddFlags(fss.FlagSet("leader"))
	s.HTTPOptions.AddFlags(fss.FlagSet("http"))

	return fss
}
