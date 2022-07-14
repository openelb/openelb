package server

import (
	"github.com/openelb/openelb/pkg/manager/client"
	"github.com/openelb/openelb/pkg/server/internal/endpoint"
	"github.com/openelb/openelb/pkg/server/internal/lib"
	"github.com/openelb/openelb/pkg/server/internal/service"
	"github.com/openelb/openelb/pkg/server/options"
)

func SetupHTTPServer(stopCh <-chan struct{}, opts *options.Options) error {
	bgpConfService := service.NewBgpConfService(client.Client)
	bgpPeerService := service.NewBgpPeerService(client.Client)
	eipService := service.NewEipService(client.Client)

	server := lib.NewHTTPServer([]lib.Endpoints{
		endpoint.NewBgpConfEndpoints(bgpConfService),
		endpoint.NewBgpPeerEndpoints(bgpPeerService),
		endpoint.NewEipEndpoints(eipService),
	}, *opts)
	return server.ListenAndServe(stopCh)
}
