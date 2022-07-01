package server

import (
	"github.com/openelb/openelb/pkg/manager/client"
	"github.com/openelb/openelb/pkg/server/internal/endpoint"
	"github.com/openelb/openelb/pkg/server/internal/kubernetes"
	"github.com/openelb/openelb/pkg/server/internal/lib"
	"github.com/openelb/openelb/pkg/server/internal/service"
	"github.com/openelb/openelb/pkg/server/options"
)

func SetupHTTPServer(opts *options.Options) error {
	bgpStore := kubernetes.NewBgpStore(client.Client)
	bgpConfService := service.NewBgpConfService(bgpStore)
	bgpPeerService := service.NewBgpPeerService(bgpStore)
	server := lib.NewHTTPServer([]lib.Endpoints{
		endpoint.NewBgpConfEndpoints(bgpConfService),
		endpoint.NewBgpPeerEndpoints(bgpPeerService),
	}, *opts)
	return server.ListenAndServe()
}
