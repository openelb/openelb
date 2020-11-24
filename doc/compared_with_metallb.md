# Compared to MetalLB

> English | [中文](zh/compared_with_metallb.md)


* Cloud Native Architecture

In Porter, you can use CRD to configure both address management and BGP configuration management, and you can view resource status, which is simple and extensible. 

In MetalLB, they are all configured via configmap, and you have to check monitoring or logging to see their status.

* Address Management

In Porter, addresses are managed via the Eip CRD, which defines the child resource status to store the status of the address assignment, so that there is no conflict between copies of the address assignment, and the programming logic is simple.

* Publishing routes with gobgp

Unlike MetalLB's implementation of the BGP protocol itself, Porter uses [gobgp](https://github.com/osrg/gobgp/blob/master/docs/sources/lib.md) to distribute routes, which has the following advantages.
1. low development cost and supported by the gobgp community
2. can take advantage of the rich features of gobgp

* Dynamic Configuration of gobgp via BgpConf/BgpPeer CRD

When gobgp is used as a lib, the community provides an [API](https://github.com/osrg/gobgp/blob/master/api/gobgp.pb.go) based on the protobuf, which Porter also refers to when implementing the BgpConf/BgpPeer CRD , and maintain compatibility with the gobgp API.

Porter also provides status for viewing the BGP neighbor configuration, which is rich in status information.

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


* Simple architecture and low resource consumption

Porter is currently deploying Deployment only, which enables high availability through multiple replicas **Not all replicas are crashed without affecting the normal established connections**.

In BGP mode, different replicas of the Deployment will connect to the router for issuing equivalent routes, so normally we can deploy two replicas.

In layer2 mode, the replicas elect a leader via the Leader Election mechanism provided by Kuberenetes, which in turn responds to ARP/NDP.