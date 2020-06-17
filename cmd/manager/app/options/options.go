package options

import (
	bgpserver "github.com/kubesphere/porter/pkg/bgp/serverd"
	"github.com/kubesphere/porter/pkg/log"
	cliflag "k8s.io/component-base/cli/flag"
)

type PorterManagerOptions struct {
	Bgp *bgpserver.BgpOptions
	*GenericOptions
	LogOptions *log.LogOptions
}

func NewPorterManagerOptions() *PorterManagerOptions {
	return &PorterManagerOptions{
		Bgp:            bgpserver.NewBgpOptions(),
		GenericOptions: NewGenericOptions(),
		LogOptions:     log.NewLogOptions(),
	}
}

func (s *PorterManagerOptions) Validate() []error {
	var errs []error
	return errs
}

func (s *PorterManagerOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	s.Bgp.AddFlags(fss.FlagSet("bgp"))
	s.GenericOptions.AddFlags(fss.FlagSet("generic"))
	s.LogOptions.AddFlags(fss.FlagSet("log"))

	return fss
}
