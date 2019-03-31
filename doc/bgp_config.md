# BGP Config介绍

Porter使用了[gobgp](https://github.com/osrg/gobgp)来与外部路由器做路由信息交换，启动BGP需要一些配置信息（样例在[config/bgp](https://github.com/kubesphere/porter/blob/master/config/bgp/config.toml)中），目前用到的参数不多，下面简单介绍如何配置插件用到的BGP服务端。

## 示例配置文件
```toml
[global.config]
    as = 65000
    router-id = "192.168.98.111"
    port = 17900
[porter-config]
    using-port-forward =true
[[neighbors]]
    [neighbors.config]
        neighbor-address = "192.168.98.5"
        peer-as = 65001
    [neighbors.add-paths.config]
        send-max = 8
```
配置文件是一个JSON文件，JSON文件有多种表达方式，`toml`、`yaml`和`json`是常见的三种形式，gobgp默认是toml，也可以根据需要转换形式。修改`porter`的进程参数`-t`来指定配置文件的格式。如：
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

## 全局配置
> 修改`global.config`来修改全局参数

1. `as`是集群所在自治域，必须和相连的路由器所在自治域不同，相同会导致路由无法正确传输，具体原因涉及到`EBGP`和`IBGP`两种协议的不同，这里不多加赘述。
2. `route-id`表示集群的id，一般取k8s主节点主网卡的ip。
3. `port`是gobgp监听的端口，默认是179。由于calico也使用了BGP，并且占用了179端口，所以这里必须指定另外的端口。如果集群的路由器不支持非179以外的端口，那么需要在port所在节点开启端口转发，将179映射到非标准端口。

## Port配置
> 和port相关的配置

1.  `using-port-forward`开启端口转发，用于交换机不支持179以外的端口，比如思科交换机。

## 设置邻居
> 邻居即集群所在的路由器。可以添加多个邻居，大多数情况下只需配置一个。

1. `neighbor-address`是路由器所在IP地址。
2. `peer-as`是邻居所在自治域，必须与集群不同，而且还需要同路由器中配置的参数一致。 如果是私网，一般使用65000以上的自治域。
3. `send-max`指定发送路由的上限，如果要实现ECMP功能，这个值必须大于1

`porter`只使用了gobgp中的一小部分功能，如果有更多的需求，可以参考gobgp的配置文件说明<https://github.com/osrg/gobgp/blob/master/docs/sources/configuration.md>