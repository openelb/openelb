# BGP Config介绍

> [English](../bgp_config.md) | 中文

Porter使用了[gobgp](https://github.com/osrg/gobgp)来与外部路由器建立BGP连接进行路由交换。

Porter提供BgpConf和BgpPeer两个CRD用于分别配置gobgp。 这两个CRD定义参考的[gobgp的API](https://github.com/osrg/gobgp/blob/master/api/gobgp.pb.go), 具体使用可以参考[GoBGP as a Go Native BGP library](https://github.com/osrg/gobgp/blob/master/docs/sources/lib.md)

## BgpConf

BgpConf用于配置gobgp的全局配置， 所以他只会有一个起作用，目前Porter只会识别name为`default`的配置。

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

1. `as`是集群所在自治域，必须和相连的路由器所在自治域不同，相同会导致路由无法正确传输。
2. `routerId` 表示集群的Id，一般取Kubernetes主节点主网卡的IP。如果你不指定，那么Porter会选择所在节点的第一个IP作为routerId。
3. `listenPort`是gobgp监听的端口，默认是179。由于Calico也使用了BGP，并且占用了179端口，所以这里必须指定另外的端口。

### 指定gobgp监听IP地址

通过`ListenAddresses`指定gobgp监听的IP地址。

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

BgpPeer用于配置gobgp的neighbor， 它可以存在多个，具体配置根据自己网络环境调整。

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

1. `conf.neighborAddress`是路由器所在IP地址。
2. `conf.peerAs`是路由器的自治域，必须与集群不同，而且还需要同路由器中配置的参数一致。 如果是私网，一般使用65000以上的自治域。

### 指定sendMax

`sendMax`用于表示gobgp发送ECMP路由时，最大等价路由数是多少， 默认为10。 可以通过以下配置指定
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

### 指定nodeSelector

当创建BgpPeer之后， 默认所有的Porter Manager副本都会响应这个配置，并与它建立连接，但是在某些场景下， Kubernetes集群节点部署在不同的路由器下，这个时候需要通过设置`nodeSelector`指定gobgp与路由器之间建立连接的关系
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
以上配置表示node4上的Porter Manager才会与172.22.0.2建立BGP连接
