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
	r.Post("/eip", e.create)
	r.Get("/eip/{name}", e.get)
	r.Get("/eip", e.list)
	r.Delete("/eip", e.delete)
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
			return nil, e.handler.Create(r.Context(), &eip)
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

func (e *eipRouter) delete(w http.ResponseWriter, r *http.Request) {
	var eip v1alpha2.Eip
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return nil, e.handler.Delete(r.Context(), &eip)
		},
		ReqBody:    &eip,
		StatusCode: http.StatusNoContent,
	})
}
