package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/server/internal/handler"
	"github.com/openelb/openelb/pkg/server/internal/lib"
)

type eipRouter struct {
	handler handler.EipHandler
}

func (e *eipRouter) Register(r chi.Router) {
	r.Post("/apis/v1/eip", e.create)
	r.Get("/apis/v1/eip/{name}", e.get)
	r.Get("/apis/v1/eip", e.list)
	r.Patch("/apis/v1/eip/{name}", e.patch)
	r.Put("/apis/v1/eip/{name}", e.update)
	r.Delete("/apis/v1/eip/{name}", e.delete)
}

// NewEipRouter returns a new instance of eipRouter which implements the
// Router interface. This is used to register the endpoints to the router.
func NewEipRouter(handler handler.EipHandler) *eipRouter {
	return &eipRouter{
		handler,
	}
}

func (e *eipRouter) create(w http.ResponseWriter, r *http.Request) {
	var eip v1alpha2.Eip
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return e.handler.Create(r.Context(), &eip)
		},
		ReqBody:    &eip,
		StatusCode: http.StatusCreated,
	})
}

func (e *eipRouter) get(w http.ResponseWriter, r *http.Request) {
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return e.handler.Get(r.Context(), chi.URLParam(r, "name"))
		},
		StatusCode: http.StatusOK,
	})
}

func (e *eipRouter) list(w http.ResponseWriter, r *http.Request) {
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return e.handler.List(r.Context())
		},
		StatusCode: http.StatusOK,
	})
}

func (e *eipRouter) patch(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var patch []byte
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return e.handler.Patch(r.Context(), name, patch)
		},
		ReqBody:    &patch,
		StatusCode: http.StatusOK,
	})
}

func (b *eipRouter) update(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var eip v1alpha2.Eip
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return b.handler.Update(r.Context(), name, &eip)
		},
		ReqBody: &eip,
		StatusCode: http.StatusOK,
	})
}

func (e *eipRouter) delete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return e.handler.Delete(r.Context(), name)
		},
		StatusCode: http.StatusNoContent,
	})
}
