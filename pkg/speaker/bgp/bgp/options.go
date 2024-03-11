package bgp

import (
	"github.com/go-logr/logr"
	"github.com/osrg/gobgp/pkg/server"
	"github.com/spf13/pflag"
)

type BgpOptions struct {
	GrpcHosts string `long:"api-hosts" description:"specify the hosts that gobgpd listens on" default:":50051"`
}

func NewBgpOptions() *BgpOptions {
	return &BgpOptions{
		GrpcHosts: ":50051",
	}
}

func (options *BgpOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&options.GrpcHosts, "api-hosts", options.GrpcHosts, "specify the hosts that gobgpd listens on")
}

type Bgp struct {
	bgpServer *server.BgpServer
	rack      string
	log       logr.Logger
}
