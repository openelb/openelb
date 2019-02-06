package routes

import (
	"context"
	"net"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	bgp "github.com/kubesphere/porter/pkg/bgp/serverd"
	"github.com/magicsong/porter/pkg/bgp/apiutil"
	api "github.com/osrg/gobgp/api"
	"github.com/vishvananda/netlink"
)

var mainLink netlink.Link

func init() {
	link, err := netlink.LinkByName("eth0")
	if err != nil {
		panic(err)
	}
	mainLink = link
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
		Family: &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST},
		Nlri:   nlri,
		Pattrs: attrs,
	}
}

func IsRouteAdded(ip string, prefix uint32) bool {
	lookup := &api.TableLookupPrefix{
		Prefix: ip,
	}
	listPathRequest := &api.ListPathRequest{
		TableType: api.TableType_GLOBAL,
		Family:    &api.Family{Afi: api.Family_AFI_IP, Safi: api.Family_SAFI_UNICAST},
		Prefixes:  []*api.TableLookupPrefix{lookup},
	}
	var result bool
	fn := func(d *api.Destination) {
		result = true
	}
	err := bgp.GetServer().ListPath(context.Background(), listPathRequest, fn)
	if err != nil {
		panic(err)
	}
	return result
}

func ReconcileRoutes(ip string, nexthops []string) (toAdd []string, toDelete []string, err error) {
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
	fn := func(d *api.Destination) {
		for _, path := range d.Paths {
			attrInterfaces, _ := apiutil.UnmarshalPathAttributes()
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

	err = bgp.GetServer().ListPath(context.Background(), listPathRequest, fn)
	return
}
func AddMultiRoutes(ip string, prefix uint32, nexthops []string) error {
	s := bgp.GetServer()
	for _, nexthop := range nexthops {
		apipath := toAPIPath(ip, prefix, nexthop)
		_, err := s.AddPath(context.Background(), &api.AddPathRequest{
			Path: apipath,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteMultiRoutes(ip string, prefix uint32, nexthops []string) error {
	s := bgp.GetServer()
	for _, nexthop := range nexthops {
		apipath := toAPIPath(ip, prefix, nexthop)
		err := s.DeletePath(context.Background(), &api.DeletePathRequest{
			Path: apipath,
		})
		if err != nil {
			return err
		}
	}
	return nil
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

func AddRoute(ip string, prefix uint32, nexthops []string) error {
	toAdd, toDelete, err := ReconcileRoutes(ip, nexthops)
	if err != nil {
		return err
	}
	err = AddMultiRoutes(ip, prefix, toAdd)
	if err != nil {
		return err
	}
	err = DeleteMultiRoutes(ip, prefix, toDelete)
	if err != nil {
		return err
	}
	return nil
}

func DeleteRoutes(ip string, nexthops []string) error {
	return DeleteMultiRoutes(ip, 32, nexthops)
}
