# 与MetalLB相比

> [English](../compared_with_metallb.md) | 中文

* 云原生架构

在Porter中， 不管是地址管理，还是BGP配置管理， 你都可以使用CRD来配置， 并且可以查看资源状态， 简单且扩展灵活。 

在MetalLB中， 他们都是通过configmap来配置， 感知它们的状态都得通过查看监控或者日志。

* 地址管理

在Porter中， 通过Eip CRD来管理地址， 它定义子资源status来存储地址分配状态， 这样就不会存在分配地址时各副本发生冲突， 编程时逻辑也会简单。

* 使用gobgp发布路由

不同于MetalLB自己实现BGP协议， Porter采用[gobgp](https://github.com/osrg/gobgp/blob/master/docs/sources/lib.md)来发布路由，这样做的好处如下：
1. 开发成本低，且有gobgp社区支持
2. 可以利用gobgp丰富特性

* 通过BgpConf/BgpPeer CRD动态配置gobgp

gobgp作为lib使用时， 社区提供了基于protobuf的[API](https://github.com/osrg/gobgp/blob/master/api/gobgp.pb.go)， Porter在实现BgpConf/BgpPeer CRD时也是参照该API，并保持兼容。

同时， Porter也提供status用于查看BGP neighbor配置， 状态信息丰富：
```bash
root@node1:/tmp# kubectl get bgppeers.network.kubesphere.io bgppeer-sample -o yaml
apiVersion: network.kubesphere.io/v1alpha2
kind: BgpPeer
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"network.kubesphere.io/v1alpha2","kind":"BgpPeer","metadata":{"annotations":{},"name":"bgppeer-sample"},"spec":{"conf":{"neighborAddress":"172.22.0.2","peerAs":50000}}}
  creationTimestamp: "2020-11-20T09:00:52Z"
  finalizers:
  - finalizer.lb.kubesphere.io/v1alpha1
  generation: 5
  name: bgppeer-sample
  resourceVersion: "6634958"
  selfLink: /apis/network.kubesphere.io/v1alpha2/bgppeers/bgppeer-sample
  uid: 70bdd404-b01a-46ec-a7fe-e307a3fa41e8
spec:
  conf:
    neighborAddress: 172.22.0.2
    peerAs: 50000
  nodeSelector:
    matchLabels:
      kubernetes.io/hostname: node4
status:
  nodesPeerStatus:
    node4:
      peerState:
        messages:
          received:
            keepalive: "170"
            open: "1"
            total: "173"
            update: "2"
          sent:
            keepalive: "149"
            open: "1"
            total: "150"
        neighborAddress: 172.22.0.2
        peerAs: 50000
        peerType: 1
        queues: {}
        routerId: 198.51.100.1
        sessionState: ESTABLISHED
      timersState:
        downtime: "2020-11-24T04:51:53Z"
        keepaliveInterval: "30"
        negotiatedHoldTime: "90"
        uptime: "2020-11-24T04:51:53Z"
```


* 架构简单，资源占用少

Porter目前只用部署Deployment即可， 通过多副本实现高可用，**非全部副本crash之后并不不会影响正常已建立连接**。

BGP模式下， Deployment不同副本都会与路由器建立连接用于发布等价路由， 所以正常情况下我们部署两个副本即可。
在layer2模式下，不同副本之间通过Kuberenetes 提供的Leader Election机制选举leader， 进而应答ARP/NDP。
