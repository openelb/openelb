# Eip Configuration

> [English](../eip_config.md) | 中文

Eip用于配置IP地址段，Porter会将其分配给LoadBalancer Service，后续通过`BGP/ARP/NDP`等协议发布Eip路由。Porter目前只支持IPv4地址, 对IPv6地支持也会马上完成。

下面Eip的例子展示了全部可用配置字段以及状态字段说明

```yaml
apiVersion: network.kubesphere.io/v1alpha2
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    address: 192.168.0.0/24
    protocol: layer2
    interface: eth0
    disable: false
status:
    occupied: false
    usage: 1
    poolSize: 256
    used: 
      "192.168.0.1": "default/test-svc"
    firstIP: 192.168.0.0
    lastIP: 192.168.0.255
    ready: true
    v4: true
```

## spec字段解释

* address

`address`用于描述IP地址范围， 它可以有如下三种格式

```yaml
- ip        e.g.  192.168.0.1
- ip/net    e.g.  192.168.0.0/24
- ip1-ip2   e.g.  192.168.0.1-192.168.0.10
```

**Note：IP地址段不可与其他已经创建的Eip重叠，否则创建资源会报错**

* protocol

`protocol`用于描述使用何种协议发布路由，合法的值有`layer2` 和 `bgp`. 当值为空时， 模式协议为`bgp`.

* interface

`interface`在`protocol`为`layer2`时才有意义， 它用于指示Porter在哪块网卡上监听ARP/NDP请求。

当Kubernetes集群中每个节点中网卡名字不同时， 你可以通过这种语法`interface: can_reach:192.168.1.1`指定网卡。 以上例子中， Porter通过查找到192.168.1.1的路由，获取路由中的第一块网卡。

* disable

当值为`true`时， 新创建LoadBalancer Service时Porter将不会从这个Eip中分配地址， 但是不会影响已经创建的Service。

## status字段解释

* occupied

此字段用于表示Eip中地址是否被分配使用完。

* usage 和 used

`usage`用于表示Eip中已经分配了多少个地址； `used`用于表示哪个地址正在被哪个Service使用， key为IP地址，value为Service的`Namespace/Name`.

* poolSize

此字段用于表示Eip中总共有多少个地址

* firstIP

此字段用于表示Eip中第一个IP地址

* lastIP

此字段用于表示Eip中最后一个IP地址

* v4

此字段用于表示Eip的地址协议族

* ready

此字段用于表示Eip关联的BGP/ARP/NDP相关程序是否初始化完毕