# Introduction to BGP Config

> English | [中文](zh/bgp_config.md)

Porter uses [gobgp](https://github.com/osrg/gobgp) to establish a BGP connection with an external router for route publishing.

Porter provides two CRDs, BgpConf and BgpPeer, for configuring gobgp respectively. The CRDs are defined in the reference [API for gobgp] (https://github.com/osrg/gobgp/blob/master/api/gobgp.pb.go), which can be used as follows Reference [GoBGP as a Go Native BGP library](https://github.com/osrg/gobgp/blob/master/docs/sources/lib.md)

## BgpConf

BgpConf is used to configure the global configuration of gobgp, so only one of these will work, and Porter currently only recognizes configurations with name `default`.

```yaml
apiVersion: network.kubesphere.io/v1alpha2
kind: BgpConf
metadata:
  #The porter only recognizes configurations with default names;
  #configurations with other names are ignored.
  name: default
spec:
  as: 50001
  listenPort: 17900
  #Modify the router id as you see fit, if it is not specified
  #then the porter will use the node ip as the router id.
  routerId: 172.22.0.10
```

1. `as` is the number of Autonomous System, which must be different from the Autonomous System of the connected routers, the same will cause the route to be incorrectly transmitted.
2. `routerId` denotes the cluster's Id, usually taking the IP of the master NIC of the Kubernetes master node. If you don't specify it, Porter will select the first IP of the node as the routerId.
3. `listenPort` is the port on which gobgp listens, which defaults to 179. Since Calico also uses BGP and occupies port 179, a different port must be specified here.

### Specify gobgp to listen to IP addresses

Specify the IP address that gobgp listens to via `ListenAddresses`.

```yaml
apiVersion: network.kubesphere.io/v1alpha2
kind: BgpConf
metadata:
  #The porter only recognizes configurations with default names;
  #configurations with other names are ignored.
  name: default
spec:
  as: 50001
  listenPort: 17900
  #Modify the router id as you see fit, if it is not specified
  #then the porter will use the node ip as the router id.
  routerId: 172.22.0.10
  ListenAddresses:
    - 172.22.0.10
```

## BgpPeer

BgpPeer is used to configure gobgp's neighbor, which can exist in multiple locations, depending on your network environment.

```yaml
apiVersion: network.kubesphere.io/v1alpha2
kind: BgpPeer
metadata:
  name: bgppeer-sample
spec:
  conf:
    peerAs: 50000
    neighborAddress: 172.22.0.2
```

1. `conf.neighborAddress` is the IP address of the router.
2. `conf.peerAs` is the Autonomous System of the router and must be different from the cluster. If it is a private network, generally use an Autonomous System above 65000.

### Specify sendMax

`sendMax` is used to indicate the maximum number of equivalent routes that gobgp can send when sending ECMP routes; the default is 10. It can be specified in the following configuration

```yaml
apiVersion: network.kubesphere.io/v1alpha2
kind: BgpPeer
metadata:
  name: bgppeer-sample
spec:
  conf:
    peerAs: 50000
    neighborAddress: 172.22.0.2
  afiSafis:
    - config:
        family:
          afi: AFI_IP
          safi: SAFI_UNICAST
        enabled: true
      addPaths:
        config:
          sendMax: 10
```

### Specify nodeSelector

When BgpPeer is created, by default all replicas of Porter Manager will respond to this configuration and establish a connection with it, but in some scenarios where Kubernetes cluster nodes are deployed under different routers, you need to specify the relationship between gobgp and the router to establish a connection by setting `nodeSelector`.

```yaml
apiVersion: network.kubesphere.io/v1alpha2
kind: BgpPeer
metadata:
  name: bgppeer-sample
spec:
  conf:
    peerAs: 50000
    neighborAddress: 172.22.0.2
  nodeSelector:
      matchLabels:
        kubernetes.io/hostname: node4
```

The above configuration means that only Porter Manager on node4 will establish a BGP connection with 172.22.0.2.