package serverd

import (
	api "github.com/osrg/gobgp/api"
	"golang.org/x/net/context"
)

// struct for container bgp:config.
// Configuration parameters relating to the global BGP router.
type BgpConfSpec struct {
	// original -> bgp:as
	// bgp:as's original type is inet:as-number.
	// Local autonomous system number of the router.  Uses
	// the 32-bit as-number type from the model in RFC 6991.
	As uint32 `mapstructure:"as" json:"as,omitempty"`
	// original -> bgp:router-id
	// bgp:router-id's original type is inet:ipv4-address.
	// Router id of the router, expressed as an
	// 32-bit value, IPv4 address.
	RouterId string `mapstructure:"router-id" json:"router-id,omitempty"`
	// original -> gobgp:port
	Port int32 `mapstructure:"port" json:"port,omitempty"`
}

func (server *BgpServer) AddBgpGlobalConfig(global *BgpConfSpec) error {
	if err := server.bgpServer.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: &api.Global{
			As:         global.As,
			RouterId:   global.RouterId,
			ListenPort: global.Port,
		},
	}); err != nil {
		return err
	}

	return nil
}
