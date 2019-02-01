package routes

import (
	"context"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	bgp "github.com/kubesphere/porter/pkg/bgp/serverd"
	"github.com/kubesphere/porter/pkg/util"
	api "github.com/osrg/gobgp/api"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
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
func AddRoute(ip string, prefix uint32, nexthop string) error {
	s := bgp.GetServer()
	if IsRouteAdded(ip, prefix) {
		log.Infoln("Detect route is existing ")
		return nil
	}
	apipath := toAPIPath(ip, prefix, nexthop)
	_, err := s.AddPath(context.Background(), &api.AddPathRequest{
		Path: apipath,
	})
	return err
}

func deleteRoute(ip string, prefix uint32, nexthop string) error {
	s := bgp.GetServer()
	apipath := toAPIPath(ip, prefix, nexthop)
	return s.DeletePath(context.Background(), &api.DeletePathRequest{
		Path: apipath,
	})
}

func AddVIP(ip string, prefix uint32) error {
	addr, err := netlink.ParseAddr(util.ToCommonString(ip, prefix))
	if err != nil {
		return err
	}
	if isAddrExist(addr) {
		log.Info("detect vip in eth0, creating vip skipped")
		return nil
	}
	return netlink.AddrAdd(mainLink, addr)
}

func DeleteVIP(ip string, prefix uint32) error {
	addr, err := netlink.ParseAddr(util.ToCommonString(ip, prefix))
	if err != nil {
		return err
	}
	if !isAddrExist(addr) {
		log.Info("detect no vip in eth0, deleting vip skipped")
		return nil
	}
	return netlink.AddrDel(mainLink, addr)
}

func isAddrExist(find *netlink.Addr) bool {
	addrs, err := netlink.AddrList(mainLink, unix.AF_INET)
	if err != nil {
		log.Errorf("Failed to get addrs of link,err:%s", err.Error())
		return false
	}
	for _, addr := range addrs {
		if addr.Equal(*find) {
			return true
		}
	}
	return false
}
