package options

import (
	"flag"
	"strings"

	"github.com/openelb/openelb/pkg/manager"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
)

type OpenELBManagerOptions struct {
	*manager.GenericOptions
}

func NewOpenELBManagerOptions() *OpenELBManagerOptions {
	return &OpenELBManagerOptions{
		GenericOptions: manager.NewGenericOptions(),
	}
}

func (s *OpenELBManagerOptions) Validate() []error {
	var errs []error
	return errs
}

func (s *OpenELBManagerOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}
	s.GenericOptions.AddFlags(fss.FlagSet("generic"))

	kfs := fss.FlagSet("klog")
	local := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = strings.Replace(fl.Name, "_", "-", -1)
		kfs.AddGoFlag(fl)
	})

	return fss
}
