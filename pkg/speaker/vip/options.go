package vip

import (
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/util"
	"github.com/spf13/pflag"
)

type VipOptions struct {
	EnableVIP       bool
	ConfigName      string
	ConfigNamespace string
	HealthPort      int
}

func NewVipOptions() *VipOptions {
	return &VipOptions{
		EnableVIP:       false,
		HealthPort:      8080,
		ConfigName:      constant.OpenELBVipConfigMap,
		ConfigNamespace: util.EnvNamespace(),
	}
}

func (v *VipOptions) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&v.EnableVIP, "enable-keepalived-vip", v.EnableVIP, "specify whether to start keepalived-vip")
	fs.IntVar(&v.HealthPort, "health-port", v.HealthPort, "The HTTP port to use for health checks")
	fs.StringVar(&v.ConfigName, "configmap-name", v.ConfigName, "specify the name of the keepalived-vip configmap")
	fs.StringVar(&v.ConfigNamespace, "configmap-namespace", v.ConfigNamespace, "specify the namespace of the keepalived-vip configmap")
}
