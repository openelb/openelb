package serverd

import (
	api "github.com/osrg/gobgp/api"
	"github.com/osrg/gobgp/pkg/server"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
)

//RunAlone is used for test
func RunAlone(ready chan<- interface{}) {
	maxSize := 256 << 20
	grpcOpts := []grpc.ServerOption{grpc.MaxRecvMsgSize(maxSize), grpc.MaxSendMsgSize(maxSize)}

	bgpServer := server.NewBgpServer(server.GrpcListenAddress(":50052"), server.GrpcOption(grpcOpts))
	go bgpServer.Serve()
	if err := bgpServer.StartBgp(context.Background(), &api.StartBgpRequest{
		Global: &api.Global{
			As:               65003,
			RouterId:         "10.0.255.254",
			ListenPort:       -1, // gobgp won't listen on tcp:179
			UseMultiplePaths: true,
		},
	}); err != nil {
		log.Fatal(err)
	}
	ready <- 0
	select {}
}
