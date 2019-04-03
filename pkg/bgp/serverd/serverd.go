//
// Copyright (C) 2014-2017 Nippon Telegraph and Telephone Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package serverd

import (
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/kubesphere/porter/pkg/bgp/config"
	"github.com/kubesphere/porter/pkg/bgp/table"
	"github.com/kubesphere/porter/pkg/nettool"
	"github.com/kubesphere/porter/pkg/util"
	api "github.com/osrg/gobgp/api"
	"github.com/osrg/gobgp/pkg/packet/bgp"
	"github.com/osrg/gobgp/pkg/server"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"gopkg.in/fsnotify.v1"
)

type StartOption struct {
	ConfigFile      string `short:"f" long:"config-file" description:"specifying a config file"`
	ConfigType      string `short:"t" long:"config-type" description:"specifying config type (toml, yaml, json)" default:"toml"`
	GrpcHosts       string `long:"api-hosts" description:"specify the hosts that gobgpd listens on" default:":50051"`
	GracefulRestart bool   `short:"r" long:"graceful-restart" description:"flag restart-state in graceful-restart capability"`
}

var bgpServer *server.BgpServer

func GetServer() *server.BgpServer {
	if bgpServer == nil {
		log.Fatalln("BGP must start before using")
	}
	return bgpServer
}

//RunAlone is used for test
func RunAlone(ready chan<- interface{}) {
	maxSize := 256 << 20
	grpcOpts := []grpc.ServerOption{grpc.MaxRecvMsgSize(maxSize), grpc.MaxSendMsgSize(maxSize)}
	log.Info("gobgpd started")
	bgpServer = server.NewBgpServer(server.GrpcListenAddress(":50052"), server.GrpcOption(grpcOpts))
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

func WatchConfigMapChange(filepath string, sigCh chan os.Signal) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalln(err)
	}
	defer watcher.Close()
	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event, "name", event.Name)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified config:", event.Name)
					sigCh <- syscall.SIGUSR1
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(path.Dir(filepath))
	if err != nil {
		log.Fatalln(err)
	}
	<-done
}
func Run(opts *StartOption, ready chan<- interface{}) {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGTERM)
	configCh := make(chan *config.BgpConfigSet)
	maxSize := 256 << 20
	grpcOpts := []grpc.ServerOption{grpc.MaxRecvMsgSize(maxSize), grpc.MaxSendMsgSize(maxSize)}

	log.Info("gobgpd started")
	bgpServer = server.NewBgpServer(server.GrpcListenAddress(opts.GrpcHosts), server.GrpcOption(grpcOpts))
	go bgpServer.Serve()
	if opts.ConfigFile == "" {
		log.Fatalln("Configfile must be non-empty")
	}

	go config.ReadConfigfileServe(opts.ConfigFile, opts.ConfigType, configCh)
	loop := func() {
		var c *config.BgpConfigSet
		for {
			select {
			case <-sigCh:
				bgpServer.StopBgp(context.Background(), &api.StopBgpRequest{})
				//clear iptables

				return
			case newConfig := <-configCh:
				var added, deleted, updated []config.Neighbor
				var addedPg, deletedPg, updatedPg []config.PeerGroup
				var updatePolicy bool

				if c == nil {
					c = newConfig
					//portforword if neccessary
					if c.PorterConfig.UsingPortForward && c.Global.Config.Port != bgp.BGP_PORT {
						if len(c.Neighbors) < 1 {
							log.Fatal("Must have at least one neighbor")
						}
						localip := util.GetOutboundIP()
						for _, nei := range c.Neighbors {
							err := nettool.AddPortForwardOfBGP(nei.Config.NeighborAddress, localip, c.Global.Config.Port)
							if err != nil {
								log.Fatalf("Error in creating iptables, %s", err.Error())
							}
						}
					}
					if err := bgpServer.StartBgp(context.Background(), &api.StartBgpRequest{
						Global: config.NewGlobalFromConfigStruct(&c.Global),
					}); err != nil {
						log.Fatalf("failed to set global config: %s", err)
					}

					if len(newConfig.Collector.Config.Url) > 0 {
						log.Fatal("collector feature is not supported")
					}

					for _, c := range newConfig.RpkiServers {
						if err := bgpServer.AddRpki(context.Background(), &api.AddRpkiRequest{
							Address:  c.Config.Address,
							Port:     c.Config.Port,
							Lifetime: c.Config.RecordLifetime,
						}); err != nil {
							log.Fatalf("failed to set rpki config: %s", err)
						}
					}
					for _, c := range newConfig.BmpServers {
						if err := bgpServer.AddBmp(context.Background(), &api.AddBmpRequest{
							Address:           c.Config.Address,
							Port:              c.Config.Port,
							Policy:            api.AddBmpRequest_MonitoringPolicy(c.Config.RouteMonitoringPolicy.ToInt()),
							StatisticsTimeout: int32(c.Config.StatisticsTimeout),
						}); err != nil {
							log.Fatalf("failed to set bmp config: %s", err)
						}
					}
					for _, c := range newConfig.MrtDump {
						if len(c.Config.FileName) == 0 {
							continue
						}
						if err := bgpServer.EnableMrt(context.Background(), &api.EnableMrtRequest{
							DumpType:         int32(c.Config.DumpType.ToInt()),
							Filename:         c.Config.FileName,
							DumpInterval:     c.Config.DumpInterval,
							RotationInterval: c.Config.RotationInterval,
						}); err != nil {
							log.Fatalf("failed to set mrt config: %s", err)
						}
					}
					p := config.ConfigSetToRoutingPolicy(newConfig)
					rp, err := table.NewAPIRoutingPolicyFromConfigStruct(p)
					if err != nil {
						log.Warn(err)
					} else {
						bgpServer.SetPolicies(context.Background(), &api.SetPoliciesRequest{
							DefinedSets: rp.DefinedSets,
							Policies:    rp.Policies,
						})
					}

					added = newConfig.Neighbors
					addedPg = newConfig.PeerGroups
					if opts.GracefulRestart {
						for i, n := range added {
							if n.GracefulRestart.Config.Enabled {
								added[i].GracefulRestart.State.LocalRestarting = true
							}
						}
					}

				} else {
					//update config
					if newConfig.PorterConfig.UsingPortForward && c.Global.Config.Port != bgp.BGP_PORT {
						if len(newConfig.Neighbors) < 1 {
							log.Fatal("Must have at least one neighbor")
						}
						localip := util.GetOutboundIP()
						for _, nei := range newConfig.Neighbors {
							err := nettool.AddPortForwardOfBGP(nei.Config.NeighborAddress, localip, c.Global.Config.Port)
							if err != nil {
								log.Fatalf("Error in creating iptables, %s", err.Error())
							}
						}
					}

					addedPg, deletedPg, updatedPg = config.UpdatePeerGroupConfig(c, newConfig)
					added, deleted, updated = config.UpdateNeighborConfig(c, newConfig)
					updatePolicy = config.CheckPolicyDifference(config.ConfigSetToRoutingPolicy(c), config.ConfigSetToRoutingPolicy(newConfig))

					if updatePolicy {
						log.Info("Policy config is updated")
						p := config.ConfigSetToRoutingPolicy(newConfig)
						rp, err := table.NewAPIRoutingPolicyFromConfigStruct(p)
						if err != nil {
							log.Warn(err)
						} else {
							bgpServer.SetPolicies(context.Background(), &api.SetPoliciesRequest{
								DefinedSets: rp.DefinedSets,
								Policies:    rp.Policies,
							})
						}
					}
					// global policy update
					if !newConfig.Global.ApplyPolicy.Config.Equal(&c.Global.ApplyPolicy.Config) {
						a := newConfig.Global.ApplyPolicy.Config
						toDefaultTable := func(r config.DefaultPolicyType) table.RouteType {
							var def table.RouteType
							switch r {
							case config.DEFAULT_POLICY_TYPE_ACCEPT_ROUTE:
								def = table.ROUTE_TYPE_ACCEPT
							case config.DEFAULT_POLICY_TYPE_REJECT_ROUTE:
								def = table.ROUTE_TYPE_REJECT
							}
							return def
						}
						toPolicies := func(r []string) []*table.Policy {
							p := make([]*table.Policy, 0, len(r))
							for _, n := range r {
								p = append(p, &table.Policy{
									Name: n,
								})
							}
							return p
						}

						def := toDefaultTable(a.DefaultImportPolicy)
						ps := toPolicies(a.ImportPolicyList)
						bgpServer.SetPolicyAssignment(context.Background(), &api.SetPolicyAssignmentRequest{
							Assignment: table.NewAPIPolicyAssignmentFromTableStruct(&table.PolicyAssignment{
								Name:     table.GLOBAL_RIB_NAME,
								Type:     table.POLICY_DIRECTION_IMPORT,
								Policies: ps,
								Default:  def,
							}),
						})

						def = toDefaultTable(a.DefaultExportPolicy)
						ps = toPolicies(a.ExportPolicyList)
						bgpServer.SetPolicyAssignment(context.Background(), &api.SetPolicyAssignmentRequest{
							Assignment: table.NewAPIPolicyAssignmentFromTableStruct(&table.PolicyAssignment{
								Name:     table.GLOBAL_RIB_NAME,
								Type:     table.POLICY_DIRECTION_EXPORT,
								Policies: ps,
								Default:  def,
							}),
						})

						updatePolicy = true

					}
					c = newConfig
				}
				for _, pg := range addedPg {
					log.Infof("PeerGroup %s is added", pg.Config.PeerGroupName)
					if err := bgpServer.AddPeerGroup(context.Background(), &api.AddPeerGroupRequest{
						PeerGroup: config.NewPeerGroupFromConfigStruct(&pg),
					}); err != nil {
						log.Warn(err)
					}
				}
				for _, pg := range deletedPg {
					log.Infof("PeerGroup %s is deleted", pg.Config.PeerGroupName)
					if err := bgpServer.DeletePeerGroup(context.Background(), &api.DeletePeerGroupRequest{
						Name: pg.Config.PeerGroupName,
					}); err != nil {
						log.Warn(err)
					}
				}
				for _, pg := range updatedPg {
					log.Infof("PeerGroup %v is updated", pg.State.PeerGroupName)
					if u, err := bgpServer.UpdatePeerGroup(context.Background(), &api.UpdatePeerGroupRequest{
						PeerGroup: config.NewPeerGroupFromConfigStruct(&pg),
					}); err != nil {
						log.Warn(err)
					} else {
						updatePolicy = updatePolicy || u.NeedsSoftResetIn
					}
				}
				for _, pg := range updatedPg {
					log.Infof("PeerGroup %s is updated", pg.Config.PeerGroupName)
					if _, err := bgpServer.UpdatePeerGroup(context.Background(), &api.UpdatePeerGroupRequest{
						PeerGroup: config.NewPeerGroupFromConfigStruct(&pg),
					}); err != nil {
						log.Warn(err)
					}
				}
				for _, dn := range newConfig.DynamicNeighbors {
					log.Infof("Dynamic Neighbor %s is added to PeerGroup %s", dn.Config.Prefix, dn.Config.PeerGroup)
					if err := bgpServer.AddDynamicNeighbor(context.Background(), &api.AddDynamicNeighborRequest{
						DynamicNeighbor: &api.DynamicNeighbor{
							Prefix:    dn.Config.Prefix,
							PeerGroup: dn.Config.PeerGroup,
						},
					}); err != nil {
						log.Warn(err)
					}
				}
				for _, p := range added {
					log.Infof("Peer %v is added", p.State.NeighborAddress)
					if err := bgpServer.AddPeer(context.Background(), &api.AddPeerRequest{
						Peer: config.NewPeerFromConfigStruct(&p),
					}); err != nil {
						log.Warn(err)
					}
				}
				for _, p := range deleted {
					log.Infof("Peer %v is deleted", p.State.NeighborAddress)
					if err := bgpServer.DeletePeer(context.Background(), &api.DeletePeerRequest{
						Address: p.State.NeighborAddress,
					}); err != nil {
						log.Warn(err)
					}
				}
				for _, p := range updated {
					log.Infof("Peer %v is updated", p.State.NeighborAddress)
					if u, err := bgpServer.UpdatePeer(context.Background(), &api.UpdatePeerRequest{
						Peer: config.NewPeerFromConfigStruct(&p),
					}); err != nil {
						log.Warn(err)
					} else {
						updatePolicy = updatePolicy || u.NeedsSoftResetIn
					}
				}

				if updatePolicy {
					if err := bgpServer.ResetPeer(context.Background(), &api.ResetPeerRequest{
						Address:   "",
						Direction: api.ResetPeerRequest_IN,
						Soft:      true,
					}); err != nil {
						log.Warn(err)
					}
				}
				ready <- 0
			}
		}
	}

	loop()
}
