package options

import (
	"flag"
	"strings"

	"github.com/openelb/openelb/pkg/log"
	"github.com/openelb/openelb/pkg/speaker/bgp/bgp"
	"github.com/openelb/openelb/pkg/speaker/layer2"
	"github.com/openelb/openelb/pkg/speaker/vip"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
)

type OpenELBSpeakerOptions struct {
	MetricsAddr string
	Bgp         *bgp.BgpOptions
	Layer2      *layer2.Options
	Vip         *vip.VipOptions
	LogOptions  *log.Options
}

func NewOpenELBSpeakerOptions() *OpenELBSpeakerOptions {
	return &OpenELBSpeakerOptions{
		MetricsAddr: ":50053",
		Bgp:         bgp.NewBgpOptions(),
		Layer2:      layer2.NewOptions(),
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
	s.Layer2.AddFlags(fss.FlagSet("layer2"))
	s.Vip.AddFlags(fss.FlagSet("vip"))
	s.LogOptions.AddFlags(fss.FlagSet("log"))

	fs := fss.FlagSet("generic")
	fs.StringVar(&s.MetricsAddr, "metrics-addr", s.MetricsAddr, "The address the metric endpoint binds to.")

	kfs := fss.FlagSet("klog")
	local := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *flag.Flag) {
		fl.Name = strings.Replace(fl.Name, "_", "-", -1)
		kfs.AddGoFlag(fl)
	})

	return fss
}
