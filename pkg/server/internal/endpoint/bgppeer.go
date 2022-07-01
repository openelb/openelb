package endpoint

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/openelb/openelb/api/v1alpha2"
	"github.com/openelb/openelb/pkg/server/internal/lib"
	"github.com/openelb/openelb/pkg/server/internal/service"
)

type bgpPeerEndpoints struct {
	service service.BgpPeerService
}

func (b *bgpPeerEndpoints) Register(r chi.Router) {
	r.Get("/bgppeer/{name}", b.get)
	r.Get("/bgppeer", b.list)
	r.Post("/bgppeer", b.create)
	r.Delete("/bgppeer", b.delete)
}

// NewBgpPeerEndpoints returns a new instance of bgpPeerEndpoints which
// implements the endpoints interface. This is used to register the endpoints to
// the router.
func NewBgpPeerEndpoints(service service.BgpPeerService) *bgpPeerEndpoints {
	return &bgpPeerEndpoints{
		service: service,
	}
}

func (b *bgpPeerEndpoints) create(w http.ResponseWriter, r *http.Request) {
	var bgpPeer v1alpha2.BgpPeer
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return nil, b.service.Create(r.Context(), &bgpPeer)
		},
		ReqBody:    &bgpPeer,
		StatusCode: http.StatusCreated,
	})
}

func (b *bgpPeerEndpoints) get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return b.service.Get(r.Context(), name)
		},
		StatusCode: http.StatusOK,
	})
}

func (b *bgpPeerEndpoints) list(w http.ResponseWriter, r *http.Request) {
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return b.service.List(r.Context())
		},
		StatusCode: http.StatusOK,
	})
}

func (b *bgpPeerEndpoints) delete(w http.ResponseWriter, r *http.Request) {
	var bgpPeer v1alpha2.BgpPeer
	lib.ServeRequest(lib.InboundRequest{
		W: w,
		R: r,
		EndpointLogic: func() (interface{}, error) {
			return nil, b.service.Delete(r.Context(), &bgpPeer)
		},
		ReqBody:    &bgpPeer,
		StatusCode: http.StatusNoContent,
	})
}
