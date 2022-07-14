package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/server/internal/handler"
	"github.com/openelb/openelb/pkg/server/internal/lib"
)

type bgpPeerRouter struct {
	handler handler.BgpPeerHandler
}

func (b *bgpPeerRouter) Register(r chi.Router) {
	r.Get("/bgppeer/{name}", b.get)
	r.Get("/bgppeer", b.list)
	r.Post("/bgppeer", b.create)
	r.Delete("/bgppeer", b.delete)
}

// NewBgpPeerRouter returns a new instance of bgpPeerRouter which
// implements the Router interface. This is used to register the endpoints to
// the router.
func NewBgpPeerRouter(handler handler.BgpPeerHandler) *bgpPeerRouter {
	return &bgpPeerRouter{
		handler: handler,
	}
}

func (b *bgpPeerRouter) create(w http.ResponseWriter, r *http.Request) {
	var bgpPeer v1alpha2.BgpPeer
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return nil, b.handler.Create(r.Context(), &bgpPeer)
		},
		ReqBody:    &bgpPeer,
		StatusCode: http.StatusCreated,
	})
}

func (b *bgpPeerRouter) get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return b.handler.Get(r.Context(), name)
		},
		StatusCode: http.StatusOK,
	})
}

func (b *bgpPeerRouter) list(w http.ResponseWriter, r *http.Request) {
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return b.handler.List(r.Context())
		},
		StatusCode: http.StatusOK,
	})
}

func (b *bgpPeerRouter) delete(w http.ResponseWriter, r *http.Request) {
	var bgpPeer v1alpha2.BgpPeer
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return nil, b.handler.Delete(r.Context(), &bgpPeer)
		},
		ReqBody:    &bgpPeer,
		StatusCode: http.StatusNoContent,
	})
}
