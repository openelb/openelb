package bgp

import (
	"sync"

	"github.com/openelb/openelb/pkg/speaker"
	api "github.com/osrg/gobgp/api"
	"github.com/osrg/gobgp/pkg/server"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ speaker.Speaker = &Bgp{}

type Client struct {
	Clientset kubernetes.Interface
}

func (c *Client) NewGoBgpd(bgpOptions *BgpOptions) *Bgp {
	maxSize := 4 << 20 //4MB
	grpcOpts := []grpc.ServerOption{grpc.MaxRecvMsgSize(maxSize), grpc.MaxSendMsgSize(maxSize)}

	bgpServer := server.NewBgpServer(server.GrpcListenAddress(bgpOptions.GrpcHosts), server.GrpcOption(grpcOpts))
	v := viper.New()
	v.SetConfigFile(bgpOptions.Conf)

	return &Bgp{
		bgpServer: bgpServer,
		client:    *c,
		v:         v,
		log:       ctrl.Log.WithName("bgpserver"),
	}
}

func (b *Bgp) run(stopCh <-chan struct{}) {
	log := ctrl.Log.WithName("gobgpd")

	log.Info("gobgpd starting")
	go b.bgpServer.Serve()
	<-stopCh
	log.Info("gobgpd ending")
	err := b.bgpServer.StopBgp(context.Background(), &api.StopBgpRequest{})
	if err != nil {
		log.Error(err, "failed to stop gobgpd")
	}
}

func (b *Bgp) Start(stopCh <-chan struct{}) error {
	go b.run(stopCh)
	return nil
}

type cache struct {
	lock  sync.Mutex
	confs map[string]interface{}
}

var (
	confs *cache
	peers *cache
)

func (c *cache) set(k string, v interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.confs[k] = v
}

func (c *cache) delete(k string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.confs, k)
}

func (c *cache) get(k string) interface{} {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.confs[k]
}

func init() {
	confs = &cache{
		lock:  sync.Mutex{},
		confs: make(map[string]interface{}),
	}
	peers = &cache{
		lock:  sync.Mutex{},
		confs: make(map[string]interface{}),
	}
}
