package options

import (
	"flag"
	"strings"

	server "github.com/openelb/openelb/pkg/server/options"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
)

type OpenELBApiServerOptions struct {
	HTTPOptions *server.Options
}

func NewOpenELBApiServerOptions() *OpenELBApiServerOptions {
	return &OpenELBApiServerOptions{
		HTTPOptions: server.NewOptions(),
	}
}

func (s *OpenELBApiServerOptions) Validate() []error {
	var errs []error
	return errs
}

func (s *OpenELBApiServerOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}
	s.HTTPOptions.AddFlags(fss.FlagSet("http"))

	kfs := fss.FlagSet("klog")
	local := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = strings.Replace(fl.Name, "_", "-", -1)
		kfs.AddGoFlag(fl)
	})

	return fss
}
