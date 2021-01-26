package options

import (
	"github.com/kubesphere/porter/pkg/leader-elector"
	"github.com/kubesphere/porter/pkg/log"
	"github.com/kubesphere/porter/pkg/manager"
	"github.com/kubesphere/porter/pkg/speaker/bgp"
	cliflag "k8s.io/component-base/cli/flag"
)

type PorterManagerOptions struct {
	Bgp *bgp.BgpOptions
	*manager.GenericOptions
	LogOptions *log.Options
	Leader     *leader.Options
}

func NewPorterManagerOptions() *PorterManagerOptions {
	return &PorterManagerOptions{
		Bgp:            bgp.NewBgpOptions(),
		GenericOptions: manager.NewGenericOptions(),
		LogOptions:     log.NewOptions(),
		Leader:         leader.NewOptions(),
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
	s.Leader.AddFlags(fss.FlagSet("leader"))

	return fss
}
