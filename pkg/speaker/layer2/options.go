package layer2

import (
	"github.com/openelb/openelb/pkg/constant"
	"github.com/openelb/openelb/pkg/util"
	"github.com/spf13/pflag"
)

type Options struct {
	EnableLayer2 bool
	NodeName     string
	BindAddr     string
	BindPort     int
	SecretKey    string
}

func NewOptions() *Options {
	return &Options{
		EnableLayer2: false,
		NodeName:     util.GetNodeName(),
		BindAddr:     "0.0.0.0",
		BindPort:     7946,
		SecretKey:    constant.Layer2MemberlistDefaultSecret,
	}
}

func (v *Options) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&v.EnableLayer2, "enable-layer2", v.EnableLayer2, "specify whether to start layer2 speaker")
	fs.StringVar(&v.NodeName, "node-name", v.NodeName, "specify node's name")
	fs.StringVar(&v.BindAddr, "bind-addr", v.BindAddr, "specify the port on which the member list listens")
	fs.IntVar(&v.BindPort, "bind-port", v.BindPort, "specify the address where the member list listens")
	fs.StringVar(&v.SecretKey, "secret", v.SecretKey, "specify the memberlist's secret")
}
