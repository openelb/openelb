package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/server/internal/handler"
	"github.com/openelb/openelb/pkg/server/internal/lib"
)

type bgpConfRouter struct {
	handler handler.BgpConfHandler
}

func (b *bgpConfRouter) Register(r chi.Router) {
	r.Post("/bgpconf", b.create)
	r.Get("/bgpconf", b.get)
	r.Delete("/bgpconf", b.delete)
}

// NewBgpConfRouter returns a new instance of bgpConfRouter which
// implements the Router interface. This is used to register the endpoints to
// the http router.
func NewBgpConfRouter(handler handler.BgpConfHandler) *bgpConfRouter {
	return &bgpConfRouter{
		handler: handler,
	}
}

func (b *bgpConfRouter) create(w http.ResponseWriter, r *http.Request) {
	var bgpConf v1alpha2.BgpConf
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return nil, b.handler.Create(r.Context(), &bgpConf)
		},
		ReqBody:    &bgpConf,
		StatusCode: http.StatusCreated,
	})
}

func (b *bgpConfRouter) get(w http.ResponseWriter, r *http.Request) {
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return b.handler.Get(r.Context())
		},
		StatusCode: http.StatusOK,
	})
}

func (b *bgpConfRouter) delete(w http.ResponseWriter, r *http.Request) {
	var bgpConf v1alpha2.BgpConf
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return nil, b.handler.Delete(r.Context(), &bgpConf)
		},
		ReqBody:    &bgpConf,
		StatusCode: http.StatusNoContent,
	})
}
