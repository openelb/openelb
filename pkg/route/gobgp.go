package route

import (
	"context"
	"net"
	"strconv"
	"strings"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/kubesphere/porter/pkg/bgp/apiutil"
	bgpserver "github.com/kubesphere/porter/pkg/bgp/serverd"
	api "github.com/osrg/gobgp/api"
	bgp "github.com/osrg/gobgp/pkg/packet/bgp"
	"github.com/osrg/gobgp/pkg/server"
)

func NewGoBGPAdvertise() Advertiser {
	return &gobgp{
		s: bgpserver.GetServer(),
	}
}

type gobgp struct {
	s *server.BgpServer
}

func GenerateIdentifier(nexthop string) uint32 {
	index := strings.LastIndex(nexthop, ".")
	n, _ := strconv.ParseUint(nexthop[index+1:], 0, 32)
	return uint32(n)
}

func toAPIPath(ip string, prefix uint32, nexthop string) *api.Path {
	nlri, _ := ptypes.MarshalAny(&api.IPAddressPrefix{
		Prefix:    ip,
		PrefixLen: prefix,
	})
	a1, _ := ptypes.MarshalAny(&api.OriginAttribute{
		Origin: 0,
	})
	a2, _ := ptypes.MarshalAny(&api.NextHopAttribute{
		NextHop: nexthop,
	})
	attrs := []*any.Any{a1, a2}
	return &api.Path{
		Family:     &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST},
		Nlri:       nlri,
		Pattrs:     attrs,
		Identifier: GenerateIdentifier(nexthop),
	}
}

func getNextHopFromPathAttributes(attrs []bgp.PathAttributeInterface) net.IP {
	for _, attr := range attrs {
		switch a := attr.(type) {
		case *bgp.PathAttributeNextHop:
			return a.Value
		case *bgp.PathAttributeMpReachNLRI:
			return a.Nexthop
		}
	}
	return nil
}

func (g *gobgp) ReconcileRoutes(ip string, prefix uint32, nexthops []string) (toAdd []string, toDelete []string, err error) {
	lookup := &api.TableLookupPrefix{
		Prefix: ip,
	}
	listPathRequest := &api.ListPathRequest{
		TableType: api.TableType_GLOBAL,
		Family:    &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST},
		Prefixes:  []*api.TableLookupPrefix{lookup},
	}
	origins := make(map[string]bool)
	news := make(map[string]bool)
	for _, item := range nexthops {
		news[item] = true
	}
	found := false
	fn := func(d *api.Destination) {
		found = true
		for _, path := range d.Paths {
			attrInterfaces, _ := apiutil.UnmarshalPathAttributes(path.Pattrs)
			nexthop := getNextHopFromPathAttributes(attrInterfaces)
			origins[nexthop.String()] = true
		}
		//compare
		for key := range origins {
			if _, ok := news[key]; !ok {
				toDelete = append(toDelete, key)
			}
		}
		for key := range news {
			if _, ok := origins[key]; !ok {
				toAdd = append(toAdd, key)
			}
		}
	}

	err = g.s.ListPath(context.Background(), listPathRequest, fn)
	if !found {
		toAdd = nexthops
	}
	return
}

func (g *gobgp) AddMultiRoutes(ip string, prefix uint32, nexthops []string) error {
	for _, nexthop := range nexthops {
		apipath := toAPIPath(ip, prefix, nexthop)
		_, err := g.s.AddPath(context.Background(), &api.AddPathRequest{
			Path: apipath,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *gobgp) DeleteMultiRoutes(ip string, prefix uint32, nexthops []string) error {
	for _, nexthop := range nexthops {
		apipath := toAPIPath(ip, prefix, nexthop)
		err := g.s.DeletePath(context.Background(), &api.DeletePathRequest{
			Path: apipath,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *gobgp) DeleteAllRoutesOfIP(ip string) error {
	lookup := &api.TableLookupPrefix{
		Prefix: ip,
	}
	listPathRequest := &api.ListPathRequest{
		TableType: api.TableType_GLOBAL,
		Family:    &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST},
		Prefixes:  []*api.TableLookupPrefix{lookup},
	}
	var errDelete error
	fn := func(d *api.Destination) {
		for _, path := range d.Paths {
			errDelete = g.s.DeletePath(context.Background(), &api.DeletePathRequest{
				Path: path,
			})
			if errDelete != nil {
				return
			}
		}
	}
	err := g.s.ListPath(context.Background(), listPathRequest, fn)
	if err != nil {
		return err
	}
	if errDelete != nil {
		return errDelete
	}
	return nil
}

