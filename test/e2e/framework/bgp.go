package framework

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/openelb/openelb/api/v1alpha2"
	gobgpapi "github.com/osrg/gobgp/api"
	"google.golang.org/grpc"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/test/e2e/framework"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OpenELBNamespace = "openelb-testns"
)

type BgpConfGlobal struct {
	AS         uint32
	ListenPort int32
	RouterID   string
	Name       string
	Client     client.Client
}

func (b *BgpConfGlobal) Create(ctx context.Context) error {
	return b.Client.Create(ctx, &v1alpha2.BgpConf{
		ObjectMeta: metav1.ObjectMeta{
			Name: b.Name,
		},
		Spec: v1alpha2.BgpConfSpec{
			As:         b.AS,
			RouterId:   b.RouterID,
			ListenPort: b.ListenPort,
		},
	})
}

func (b *BgpConfGlobal) Delete(ctx context.Context) error {
	return b.Client.Delete(ctx, &v1alpha2.BgpConf{
		ObjectMeta: metav1.ObjectMeta{
			Name: b.Name,
		},
	})
}

func (b *BgpConfGlobal) Update(ctx context.Context, listenPort int32) error {
	conf := &v1alpha2.BgpConf{}

	if err := b.Client.Get(ctx, client.ObjectKey{Name: b.Name}, conf); err != nil {
		return err
	}

	conf.Spec.ListenPort = listenPort
	b.ListenPort = listenPort

	return b.Client.Update(ctx, conf)
}

type BgpPeer struct {
	Address string
	AS      uint32
	Port    uint32
	Name    string
	Client  client.Client
	Passive bool
}

func (b *BgpPeer) Create(ctx context.Context) error {
	return b.Client.Create(ctx, &v1alpha2.BgpPeer{
		ObjectMeta: metav1.ObjectMeta{
			Name: b.Name,
		},
		Spec: v1alpha2.BgpPeerSpec{
			Conf: &v1alpha2.PeerConf{
				PeerAs:          b.AS,
				NeighborAddress: b.Address,
			},
			Transport: &v1alpha2.Transport{
				RemotePort:  b.Port,
				PassiveMode: b.Passive,
			},
		},
	})
}

func (b *BgpPeer) Delete(ctx context.Context) error {
	return b.Client.Delete(ctx, &v1alpha2.BgpPeer{
		ObjectMeta: metav1.ObjectMeta{
			Name: b.Name,
		},
	})
}

func (b *BgpPeer) Update(ctx context.Context) error {
	for i := 0; i < 3; i++ {
		bgppeer := &v1alpha2.BgpPeer{}
		if err := b.Client.Get(ctx, client.ObjectKey{Name: b.Name}, bgppeer); err != nil {
			return err
		}

		// framework.Logf("bgppeer %v", bgppeer)
		bgppeer.Spec = v1alpha2.BgpPeerSpec{
			Conf: &v1alpha2.PeerConf{
				PeerAs:          b.AS,
				NeighborAddress: b.Address,
			},
			Transport: &v1alpha2.Transport{
				RemotePort:  b.Port,
				PassiveMode: b.Passive,
			},
		}

		err := b.Client.Update(ctx, bgppeer)
		if err == nil {
			return nil
		}

		if err != nil && !errors.IsConflict(err) && !errors.IsServerTimeout(err) {
			return err
		}
	}
	return nil
}

type GobgpClient struct {
	gobgpapi.GobgpApiClient
}

func NewGobgpClient(ctx context.Context, pod *v1.Pod, port int) *GobgpClient {
	grpcOpts := []grpc.DialOption{grpc.WithBlock()}
	grpcOpts = append(grpcOpts, grpc.WithInsecure())
	target := pod.Status.PodIP + ":" + fmt.Sprintf("%d", port)
	cc, _ := context.WithTimeout(ctx, time.Second)
	conn, err := grpc.DialContext(cc, target, grpcOpts...)
	if err != nil {
		return nil
	}

	return &GobgpClient{
		GobgpApiClient: gobgpapi.NewGobgpApiClient(conn),
	}
}

func (c *GobgpClient) AddConfForGobgp(routerid string, as uint32, port int32) error {
	_, err := c.StartBgp(context.Background(), &gobgpapi.StartBgpRequest{
		Global: &gobgpapi.Global{
			As:               as,
			RouterId:         routerid,
			ListenPort:       port,
			UseMultiplePaths: true,
			GracefulRestart: &gobgpapi.GracefulRestart{
				Enabled:     true,
				RestartTime: 60,
			},
		},
	})
	return err
}

func fromAPIPath(path *gobgpapi.Path) (net.IP, error) {
	for _, attr := range path.Pattrs {
		var value ptypes.DynamicAny
		if err := ptypes.UnmarshalAny(attr, &value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal route distinguisher: %s", err)
		}

		switch a := value.Message.(type) {
		case *gobgpapi.NextHopAttribute:
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

func (c *GobgpClient) GetRoutersForGobgp(ip string) (map[string]struct{}, error) {
	listPathRequest := &gobgpapi.ListPathRequest{
		TableType: gobgpapi.TableType_GLOBAL,
		Family:    getFamily(ip),
		Prefixes: []*gobgpapi.TableLookupPrefix{
			{
				Prefix: ip,
			},
		},
	}

	nexthops := make(map[string]struct{})
	responce, err := c.ListPath(context.TODO(), listPathRequest)
	if err != nil {
		return nexthops, err
	}

	for {
		r, err := responce.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return nexthops, err
		}

		for _, path := range r.Destination.Paths {
			nexthop, _ := fromAPIPath(path)
			if _, exist := nexthops[nexthop.String()]; !exist {
				nexthops[nexthop.String()] = struct{}{}
			}
		}
	}

	return nexthops, err
}

func (c *GobgpClient) AddPeerForGobgp(address string, as uint32, port int) error {
	_, err := c.AddPeer(context.TODO(), &gobgpapi.AddPeerRequest{
		Peer: &gobgpapi.Peer{
			Conf: &gobgpapi.PeerConf{
				NeighborAddress: address,
				PeerAs:          as,
			},
			AfiSafis: []*gobgpapi.AfiSafi{
				{
					Config: &gobgpapi.AfiSafiConfig{
						Family:  getFamily(address),
						Enabled: true,
					},
					AddPaths: &gobgpapi.AddPaths{
						Config: &gobgpapi.AddPathsConfig{
							Receive: true,
							SendMax: 8,
						},
					},
					MpGracefulRestart: &gobgpapi.MpGracefulRestart{
						Config: &gobgpapi.MpGracefulRestartConfig{
							Enabled: true,
						},
					},
				},
			},
			GracefulRestart: &gobgpapi.GracefulRestart{
				Enabled:     true,
				RestartTime: 60,
			},
			Transport: &gobgpapi.Transport{
				RemotePort: uint32(port),
				//PassiveMode: true,
			},
		}})
	return err
}

func (c *GobgpClient) DeletePeerForGobgp(address string) error {
	_, err := c.DeletePeer(context.TODO(), &gobgpapi.DeletePeerRequest{
		Address: address,
	})
	return err
}

func (c *GobgpClient) GetAllPeers() ([]*gobgpapi.Peer, error) {
	peers := make([]*gobgpapi.Peer, 0)
	responce, err := c.ListPeer(context.TODO(), &gobgpapi.ListPeerRequest{})
	if err != nil {
		return peers, err
	}

	for {
		r, err := responce.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			return peers, err
		}
		peers = append(peers, r.Peer)
	}

	return peers, err
}

func getFamily(ip string) *gobgpapi.Family {
	family := &gobgpapi.Family{
		Afi:  gobgpapi.Family_AFI_IP,
		Safi: gobgpapi.Family_SAFI_UNICAST,
	}
	if net.ParseIP(ip).To4() == nil {
		family = &gobgpapi.Family{
			Afi:  gobgpapi.Family_AFI_IP6,
			Safi: gobgpapi.Family_SAFI_UNICAST,
		}
	}

	return family
}

func WaitForRouterNum(timeout time.Duration, ip string, bgpClient *GobgpClient, num int) error {
	pollFunc := func() (bool, error) {
		routers, err := bgpClient.GetRoutersForGobgp(ip)
		if err != nil {
			return false, err
		}

		klog.Infof("gobgp route %s ==> %v", ip, routers)
		return len(routers) == num, nil
	}

	return wait.PollImmediate(framework.Poll, timeout, pollFunc)
}

func WaitForBGPEstablished(timeout time.Duration, bgpClient *GobgpClient, num int) error {
	pollFunc := func() (bool, error) {
		peers, err := bgpClient.GetAllPeers()
		if err != nil {
			return false, err
		}

		return len(peers) == num, nil
	}

	return wait.PollImmediate(framework.Poll, timeout, pollFunc)
}
