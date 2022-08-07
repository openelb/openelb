package bgp

import (
	"github.com/go-logr/logr"
	api "github.com/osrg/gobgp/api"
	"github.com/osrg/gobgp/pkg/server"
	"github.com/spf13/pflag"
)

type Bgp struct {
	log       logr.Logger
	peers     []*api.Peer
	client    Client
	bgpServer *server.BgpServer
}

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
