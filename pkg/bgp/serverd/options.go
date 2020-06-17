package serverd

import (
	"github.com/go-logr/logr"
	"github.com/kubesphere/porter/pkg/nettool/iptables"
	"github.com/osrg/gobgp/pkg/server"
	"github.com/spf13/pflag"
)

type BgpOptions struct {
	GrpcHosts       string `long:"api-hosts" description:"specify the hosts that gobgpd listens on" default:":50051"`
	GracefulRestart bool   `short:"r" long:"graceful-restart" description:"flag restart-state in graceful-restart capability"`
}

func NewBgpOptions() *BgpOptions {
	return &BgpOptions{
		GrpcHosts:       ":50051",
		GracefulRestart: true,
	}
}

func (options *BgpOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&options.GrpcHosts, "api-hosts", options.GrpcHosts, "specify the hosts that gobgpd listens on")
	fs.BoolVar(&options.GracefulRestart, "r", options.GracefulRestart, "flag restart-state in graceful-restart capability")
}

type BgpServer struct {
	bgpServer  *server.BgpServer
	bgpOptions *BgpOptions
	log        logr.Logger
	bgpIptable iptables.IptablesIface
}
