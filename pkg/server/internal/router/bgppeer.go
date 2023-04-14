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
	r.Post("/apis/v1/bgp", b.create)
	r.Get("/apis/v1/bgp/{name}", b.get)
	r.Get("/apis/v1/bgp", b.list)
	r.Patch("/apis/v1/bgp/{name}", b.patch)
	r.Put("/apis/v1/bgp/{name}", b.update)
	r.Delete("/apis/v1/bgp/{name}", b.delete)
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
			return b.handler.Create(r.Context(), &bgpPeer)
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

func (b *bgpPeerRouter) patch(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var patch []byte
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return b.handler.Patch(r.Context(), name, patch)
		},
		ReqBody:    &patch,
		StatusCode: http.StatusOK,
	})
}

func (b *bgpPeerRouter) update(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var bgpPeer v1alpha2.BgpPeer
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return b.handler.Update(r.Context(), name, &bgpPeer)
		},
		ReqBody: &bgpPeer,
		StatusCode: http.StatusOK,
	})
}

func (b *bgpPeerRouter) delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return b.handler.Delete(r.Context(), name)
		},
		StatusCode: http.StatusNoContent,
	})
}
