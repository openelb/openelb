package endpoint

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/server/internal/lib"
	"github.com/openelb/openelb/pkg/server/internal/service"
)

type eipEndpoints struct {
	service service.EipService
}

func (e *eipEndpoints) Register(r chi.Router) {
	r.Post("/eip", e.create)
	r.Get("/eip/{name}", e.get)
	r.Get("/eip", e.list)
	r.Delete("/eip", e.delete)
}

// NewEipEndpoints returns a new instance of eipEndpoints which implements the
// endpoints interface. This is used to register the endpoints to the router.
func NewEipEndpoints(service service.EipService) *eipEndpoints {
	return &eipEndpoints{
		service,
	}
}

func (e *eipEndpoints) create(w http.ResponseWriter, r *http.Request) {
	var eip v1alpha2.Eip
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return nil, e.service.Create(r.Context(), &eip)
		},
		ReqBody:    &eip,
		StatusCode: http.StatusCreated,
	})
}

func (e *eipEndpoints) get(w http.ResponseWriter, r *http.Request) {
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return e.service.Get(r.Context(), chi.URLParam(r, "name"))
		},
		StatusCode: http.StatusOK,
	})
}

func (e *eipEndpoints) list(w http.ResponseWriter, r *http.Request) {
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return e.service.List(r.Context())
		},
		StatusCode: http.StatusOK,
	})
}

func (e *eipEndpoints) delete(w http.ResponseWriter, r *http.Request) {
	var eip v1alpha2.Eip
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return nil, e.service.Delete(r.Context(), &eip)
		},
		ReqBody:    &eip,
		StatusCode: http.StatusNoContent,
	})
}
