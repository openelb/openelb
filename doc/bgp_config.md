# BGP Configuration

> English | [中文](zh/bgp_config.md)

Porter uses [gobgp](https://github.com/osrg/gobgp) to exchange routing information with external routers. Starting BGP requires some configuration information (samples are in [config/bgp](https://github.com/kubesphere/porter/blob/master/config/bgp/config.toml)), there are not many parameters currently used, the following configuration introduces how to configure the BGP server used by Porter.

## Configuration
```toml
[global.config]
    as = 65000
    router-id = "192.168.98.111"
    port = 17900 ## The standard port is 179，
[porter-config]
    using-port-forward =true ## if the port is 179, please remove this line
[[neighbors]]
    [neighbors.config]
        neighbor-address = "192.168.98.5"
        peer-as = 65001
    [neighbors.add-paths.config]
        send-max = 8
```
The configuration file is a JSON file. The JSON file has multiple expressions. `Toml`,` yaml`, and `json` are the three common forms. Gobgp defaults to toml, and can also be converted as needed. Modify the args `-t` to specify the format of the configuration file. Such as:
```yaml
 - args:
        - --metrics-addr=127.0.0.1:8080
        - -f
        - /etc/config/config.yaml
        - -t
        - yaml
   command:
        - /manager
```

## Global configuration
> Modify `global.config` to specify global parameters

1. `as` is the number of Autonomous System, and it must be different from the AS number of the router.
2. `route-id` is the ID of the cluster. In general, we use the IP of the k8s master node.
3. `port` is the port that gobgp listens to, the default is 179. Because calico also uses BGP and listens the port 179, it is necessary to specify a different port here. If the router does not support ports other than 179, you need to enable port forwarding on the node where the port is located to map 179 to other port.

## Port Configuration
> Port related configuration

1.  `using-port-forward` turns on port forwarding and is used for switches that do not support ports other than 179, such as Cisco switches.

## Configuring BGP Peers
> BGP peers and neighbors mean the same thing. You can add one or more neighbors.

1. `neighbor-address` is the IP address of the router.
2. `peer-as` is the AS number of the neighbors, which must be different from the Porter's AS number. Please use private BGP number, the range is 64512 – 65534
3. `send-max` specifies the upper limit of the sending route, if you want to enable the ECMP feature, this value must be greater than 1.

`porter` only uses a small part of the functions in gobgp. For more details, please refer to [gobgp configuration](https://github.com/osrg/gobgp/blob/master/docs/sources/configuration.md)