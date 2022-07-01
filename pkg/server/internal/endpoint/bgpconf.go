package endpoint

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/server/internal/lib"
	"github.com/openelb/openelb/pkg/server/internal/service"
)

type bgpConfEndpoints struct {
	service service.BgpConfService
}

func (b *bgpConfEndpoints) Register(r chi.Router) {
	r.Post("/bgpconf", b.create)
	r.Get("/bgpconf", b.get)
	r.Delete("/bgpconf", b.delete)
}

// NewBgpConfEndpoints returns a new instance of bgpConfEndpoints which
// implements the endpoints interface. This is used to register the endpoints to
// the router.
func NewBgpConfEndpoints(service service.BgpConfService) *bgpConfEndpoints {
	return &bgpConfEndpoints{
		service: service,
	}
}

func (b *bgpConfEndpoints) create(w http.ResponseWriter, r *http.Request) {
	var bgpConf v1alpha2.BgpConf
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return nil, b.service.Create(r.Context(), &bgpConf)
		},
		ReqBody:    &bgpConf,
		StatusCode: http.StatusCreated,
	})
}

func (b *bgpConfEndpoints) get(w http.ResponseWriter, r *http.Request) {
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return b.service.Get(r.Context())
		},
		StatusCode: http.StatusOK,
	})
}

func (b *bgpConfEndpoints) delete(w http.ResponseWriter, r *http.Request) {
	var bgpConf v1alpha2.BgpConf
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return nil, b.service.Delete(r.Context(), &bgpConf)
		},
		ReqBody:    &bgpConf,
		StatusCode: http.StatusNoContent,
	})
}
