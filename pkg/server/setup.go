package server

import (
	"github.com/openelb/openelb/pkg/manager/client"
	"github.com/openelb/openelb/pkg/server/internal/handler"
	"github.com/openelb/openelb/pkg/server/internal/lib"
	"github.com/openelb/openelb/pkg/server/internal/router"
	"github.com/openelb/openelb/pkg/server/options"
)

func SetupHTTPServer(stopCh <-chan struct{}, opts *options.Options) error {
	bgpConfService := handler.NewBgpConfHandler(client.Client)
	bgpPeerService := handler.NewBgpPeerHandler(client.Client)
	eipService := handler.NewEipHandler(client.Client)

	server := lib.NewHTTPServer([]lib.Router{
		router.NewBgpConfRouter(bgpConfService),
		router.NewBgpPeerRouter(bgpPeerService),
		router.NewEipRouter(eipService),
	}, *opts)
	return server.ListenAndServe(stopCh)
}
