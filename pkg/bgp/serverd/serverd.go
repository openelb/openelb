package serverd

import (
	"fmt"
	"github.com/go-logr/logr"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/kubesphere/porter/pkg/nettool/iptables"
	api "github.com/osrg/gobgp/api"
	"github.com/osrg/gobgp/pkg/server"

	"github.com/spf13/pflag"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"strconv"
	"strings"
)

type BgpOptions struct {
	GrpcHosts       string `long:"api-hosts" description:"specify the hosts that gobgpd listens on" default:":50051"`
	GracefulRestart bool   `short:"r" long:"graceful-restart" description:"flag restart-state in graceful-restart capability"`
}

func NewBgpOptions() *BgpOptions {
	return &BgpOptions{
		GrpcHosts:       ":50051",
		GracefulRestart: true,
	}
}

func (options *BgpOptions) AddFlags(fs *pflag.FlagSet, c *BgpOptions) {
	fs.StringVar(&options.GrpcHosts, "api-hosts", c.GrpcHosts, "specify the hosts that gobgpd listens on")
	fs.BoolVar(&options.GracefulRestart, "r", false, "flag restart-state in graceful-restart capability")
}

type BgpServer struct {
	bgpServer  *server.BgpServer
	bgpOptions *BgpOptions
	Log        logr.Logger
	bgpIptable iptables.IptablesIface
}

func NewBgpServer(bgpOptions *BgpOptions) *BgpServer {
	maxSize := 256 << 20
	grpcOpts := []grpc.ServerOption{grpc.MaxRecvMsgSize(maxSize), grpc.MaxSendMsgSize(maxSize)}

	bgpServer := server.NewBgpServer(server.GrpcListenAddress(bgpOptions.GrpcHosts), server.GrpcOption(grpcOpts))
	go bgpServer.Serve()

	return &BgpServer{
		bgpServer:  bgpServer,
		bgpOptions: bgpOptions,
		bgpIptable: iptables.NewIPTables(),
	}
}

func (server *BgpServer) StopServer() error {
	return server.bgpServer.StopBgp(context.Background(), &api.StopBgpRequest{})
}

func generateIdentifier(nexthop string) uint32 {
	index := strings.LastIndex(nexthop, ".")
	n, _ := strconv.ParseUint(nexthop[index+1:], 0, 32)
	return uint32(n)
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
		Origin: 0,
	})
	a2, _ := ptypes.MarshalAny(&api.NextHopAttribute{
		NextHop: nexthop,
	})
	attrs := []*any.Any{a1, a2}

	return &api.Path{
		Family:     getFamily(ip),
		Nlri:       nlri,
		Pattrs:     attrs,
		Identifier: generateIdentifier(nexthop),
	}
}

func fromAPIPath(path *api.Path) (net.IP, error) {
	for _, attr := range path.Pattrs {
		var value ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(attr, &value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal route distinguisher: %s", err)
		}

		switch a := value.Message.(type) {
		case *api.NextHopAttribute:
			nexthop := net.ParseIP(a.NextHop).To4()
			if nexthop == nil {
				if nexthop = net.ParseIP(a.NextHop).To16(); nexthop == nil {
					return nil, fmt.Errorf("invalid nexthop address: %s", a.NextHop)
				}
			}
			return nexthop, nil
		}
	}

	return nil, fmt.Errorf("cannot find nexthop")
}

func (server *BgpServer) retriveRoutes(ip string, prefix uint32, nexthops []string) (err error, toAdd, toDelete []string) {
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
		server.Log.Info("list paths:", "paths", d.Paths)
		for _, path := range d.Paths {
			nexthop, _ := fromAPIPath(path)
			server.Log.Info("path nexthop", "nexthop", nexthop)
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

	err = server.bgpServer.ListPath(context.Background(), listPathRequest, fn)
	if err != nil {
		return
	}
	if !found {
		toAdd = nexthops
	}

	return
}

func (server *BgpServer) ReconcileRoutes(ip string, prefix uint32, nexthops []string) error {
	err, toAdd, toDelete := server.retriveRoutes(ip, prefix, nexthops)
	if err != nil {
		return err
	}

	server.Log.Info("update router:", "toAdd", toAdd, "toDelete", toDelete)
	err = server.addMultiRoutes(ip, prefix, toAdd)
	if err != nil {
		return err
	}
	err = server.deleteMultiRoutes(ip, prefix, toDelete)
	if err != nil {
		return err
	}
	return nil
}

func (server *BgpServer) addMultiRoutes(ip string, prefix uint32, nexthops []string) error {
	for _, nexthop := range nexthops {
		apipath := toAPIPath(ip, prefix, nexthop)
		server.Log.Info("add path:", "apiPath", apipath)
		_, err := server.bgpServer.AddPath(context.Background(), &api.AddPathRequest{
			Path: apipath,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (server *BgpServer) deleteMultiRoutes(ip string, prefix uint32, nexthops []string) error {
	for _, nexthop := range nexthops {
		apipath := toAPIPath(ip, prefix, nexthop)
		server.Log.Info("delete path:", "apiPath", apipath)
		err := server.bgpServer.DeletePath(context.Background(), &api.DeletePathRequest{
			Path: apipath,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (server *BgpServer) DeleteAllRoutesOfIP(ip string) error {
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
			errDelete = server.bgpServer.DeletePath(context.Background(), &api.DeletePathRequest{
				Path: path,
			})
			if errDelete != nil {
				return
			}
		}
	}
	err := server.bgpServer.ListPath(context.Background(), listPathRequest, fn)
	if err != nil {
		return err
	}
	if errDelete != nil {
		return errDelete
	}
	return nil
}
