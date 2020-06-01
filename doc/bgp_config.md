# BGP Configuration

> English | [中文](zh/bgp_config.md)

Porter uses [gobgp](https://github.com/osrg/gobgp) to exchange routing information with external routers. There are not many parameters currently used, the following configuration introduces how to configure the BGP server used by Porter.

## Global configuration
```yaml
apiVersion: network.kubesphere.io/v1alpha1
kind: BgpConf
metadata:
  name: bgpconf-sample
spec:
  # Add fields here
  as : 65000
  routerID : 192.168.0.2
  port: 17900
```

1. `as` is the number of Autonomous System, and it must be different from the AS number of the router.
2. `routeID` is the ID of the cluster. In general, we use the IP of the k8s master node.
3. `port` is the port that gobgp listens to, the default is 179. Because calico also uses BGP and listens the port 179, it is necessary to specify a different port here. If the router does not support ports other than 179, you need to enable port forwarding on the node where the port is located to map 179 to other port.

## Configuring BGP Peers
> BGP peers and neighbors mean the same thing. You can add one or more neighbors.
```yaml
apiVersion: network.kubesphere.io/v1alpha1
kind: BgpPeer
metadata:
  name: bgppeer-sample
spec:
  # Add fields here
  usingPortForward: true
  config:
    peerAs : 65001
    neighborAddress: 192.168.0.6
  addAaths:
    sendMax: 10
  transport:
    passiveMode: true
```

1. `neighborAddress` is the IP address of the router.
2. `peerAs` is the AS number of the neighbors, which must be different from the Porter's AS number. Please use private BGP number, the range is 64512 – 65534
3. `sendMax` specifies the upper limit of the sending route, if you want to enable the ECMP feature, this value must be greater than 1.
4. `usingPortForward` turns on port forwarding and is used for switches that do not support ports other than 179, such as Cisco switches.
5. `passiveMode` indicate porter manager connect router voluntarily.

`porter` only uses a small part of the functions in gobgp. For more details, please refer to [gobgp configuration](https://github.com/osrg/gobgp/blob/master/docs/sources/configuration.md)