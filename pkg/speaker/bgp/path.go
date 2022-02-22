package bgp

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/openelb/openelb/pkg/constant"
	api "github.com/osrg/gobgp/api"
	bgppacket "github.com/osrg/gobgp/pkg/packet/bgp"
	"hash/fnv"
	corev1 "k8s.io/api/core/v1"
	"net"
	ctrl "sigs.k8s.io/controller-runtime"
)

func getPathIdentifier(nexthop string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(nexthop))
	return h.Sum32()
}

func getFamily(ip string) *api.Family {
	family := &api.Family{
		Afi:  api.Family_AFI_IP,
		Safi: api.Family_SAFI_UNICAST,
	}
	if net.ParseIP(ip).To4() == nil {
		family = &api.Family{
			Afi:  api.Family_AFI_IP6,
			Safi: api.Family_SAFI_UNICAST,
		}
	}

	return family
}

func toAPIPath(ip string, prefix uint32, nexthop string) *api.Path {
	nlri, _ := ptypes.MarshalAny(&api.IPAddressPrefix{
		Prefix:    ip,
		PrefixLen: prefix,
	})
	a1, _ := ptypes.MarshalAny(&api.OriginAttribute{
		Origin: uint32(bgppacket.BGP_ORIGIN_ATTR_TYPE_IGP),
	})
	a2, _ := ptypes.MarshalAny(&api.NextHopAttribute{
		NextHop: nexthop,
	})
	attrs := []*any.Any{a1, a2}

	return &api.Path{
		Family:     getFamily(ip),
		Nlri:       nlri,
		Pattrs:     attrs,
		Identifier: getPathIdentifier(nexthop),
	}
}

func fromAPIPath(path *api.Path) net.IP {
	for _, attr := range path.Pattrs {
		var value ptypes.DynamicAny

		ptypes.UnmarshalAny(attr, &value)

		switch a := value.Message.(type) {
		case *api.NextHopAttribute:
			return net.ParseIP(a.NextHop)
		}
	}

	return nil
}

func (b *Bgp) retriveRoutes(ip string, prefix uint32, nexthops []string) (err error, toAdd, toDelete []string) {
	listPathRequest := &api.ListPathRequest{
		TableType: api.TableType_GLOBAL,
		Family:    getFamily(ip),
		Prefixes: []*api.TableLookupPrefix{
			&api.TableLookupPrefix{
				Prefix: ip,
			},
		},
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
			nexthop := fromAPIPath(path)
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

	err = b.bgpServer.ListPath(context.Background(), listPathRequest, fn)
	if err != nil {
		return
	}
	if !found {
		toAdd = nexthops
	}

	return
}

func (b *Bgp) ready() error {
	response, err := b.bgpServer.GetBgp(context.Background(), nil)
	if err != nil {
		return err
	}

	if response.Global.As == 0 {
		return fmt.Errorf("Bgp not ready, please config bgpconf/bgppeer")
	}

	return nil
}

func (b *Bgp) setBalancer(ip string, nexthops []string) error {
	err := b.ready()
	if err != nil {
		return err
	}

	prefix := uint32(32)
	err, toAdd, toDelete := b.retriveRoutes(ip, prefix, nexthops)
	if err != nil {
		return err
	}

	err = b.addMultiRoutes(ip, prefix, toAdd)
	if err != nil {
		return err
	}
	err = b.deleteMultiRoutes(ip, prefix, toDelete)
	if err != nil {
		return err
	}
	return nil
}

func (b *Bgp) SetBalancer(ip string, nodes []corev1.Node) error {
	var nexthops []string

	for _, node := range nodes {
		rack := ""
		if node.Labels != nil {
			rack = node.Labels[constant.OpenELBNodeRack]
		}
		if rack == b.rack || b.rack == "" {
			nexthop, err := b.getNodeNextHop(node)
			if err != nil {
				return err
			}
			nexthops = append(nexthops, nexthop)
		}
	}

	ctrl.Log.Info("bgp setBalancer", "nexthops", nexthops)

	return b.setBalancer(ip, nexthops)
}

func (b *Bgp) getNodeNextHop(node corev1.Node) (string, error) {
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address, nil
		}
	}

	return "", fmt.Errorf("node has no internal ip")
}

func (b *Bgp) addMultiRoutes(ip string, prefix uint32, nexthops []string) error {
	for _, nexthop := range nexthops {
		apipath := toAPIPath(ip, prefix, nexthop)
		_, err := b.bgpServer.AddPath(context.Background(), &api.AddPathRequest{
			Path: apipath,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Bgp) deleteMultiRoutes(ip string, prefix uint32, nexthops []string) error {
	for _, nexthop := range nexthops {
		apipath := toAPIPath(ip, prefix, nexthop)
		err := b.bgpServer.DeletePath(context.Background(), &api.DeletePathRequest{
			Path: apipath,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Bgp) DelBalancer(ip string) error {
	err := b.ready()
	if err != nil {
		return err
	}

	lookup := &api.TableLookupPrefix{
		Prefix: ip,
	}
	listPathRequest := &api.ListPathRequest{
		TableType: api.TableType_GLOBAL,
		Family:    getFamily(ip),
		Prefixes:  []*api.TableLookupPrefix{lookup},
	}
	var errDelete error
	fn := func(d *api.Destination) {
		for _, path := range d.Paths {
			errDelete = b.bgpServer.DeletePath(context.Background(), &api.DeletePathRequest{
				Path: path,
			})
			if errDelete != nil {
				return
			}
		}
	}
	err = b.bgpServer.ListPath(context.Background(), listPathRequest, fn)
	if err != nil {
		return err
	}
	if errDelete != nil {
		return errDelete
	}
	return nil
}
