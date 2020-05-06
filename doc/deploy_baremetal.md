# 物理机部署

## 安装前提
1.  物理机连接的路由器必须支持BGP协议
2.  如果需要实现路由器端的负载均衡，需要路由器支持ECMP，并包括以下特性：
    - 支持接收多个等价路由
    - 支持接收来自同一个邻居的多条等价路由
3. 如果网络架构中存在一个路由器不支持BGP（或者被禁止开启BGP）,那么需要在这个路由器上手动写EIP的nexthop路由（或者通过其他路由发现协议）

## 安装Porter
 1. 在机器上安装kubernetes
 2. 获取yaml
     ```bash
    wget https://github.com/kubesphere/porter/releases/download/v0.1.1/porter.yaml
     ```
 3. 修改yaml中一个名为bgp-cfg的configmap，按照[BGP配置教程](bgp_config.md)简单修改一些字段即可。注意要路由器的地址和AS域
 4. 安装porter到集群中
     ```bash
     kubectl apply -f porter.yaml
     ```
## 路由器配置
> 不同的路由器配置不同，这边仅列出一个样例的思科三层交换机的配置，更多的请参考[路由器配置](router_config.md)。本节以思科的[Nexus 9000 Series](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/92x/unicast/configuration/guide/b-cisco-nexus-9000-series-nx-os-unicast-routing-configuration-guide-92x/b-cisco-nexus-9000-series-nx-os-unicast-routing-configuration-guide-92x_chapter_01010.html)作为示例。


1. 以admin进入N9K配置界面。按照实际情况修改下面的配置。（注：实际输入不能有注释）

   ```
    feature bgp   ##开启BGP功能

    router bgp 65001 #设置本路由器AS域
    router-id 10.10.12.1 #设置本路由器IP
    address-family ipv4 unicast 
        maximum-paths 8 #开启ECMP，并且最多接受8个等价路由
        additional-paths send # 能够发送多个等价路由
        additional-paths receive # 能够接受多个等价路由
    neighbor 10.10.12.5 #邻居IP
        remote-as 65000 #邻居AS，必须和本机AS不同
        timers 10 30
        address-family ipv4 unicast
        route-map allow in #允许导入系统路由表
        route-map allow out #允许导出路由表到系统
        soft-reconfiguration inbound always # 自动更新邻居状态
        capability additional-paths receive # 开启接受该邻居多条等价路由的能力
    ```

2. 配置完成之后，查看邻居状态为`Established`即可。`show bgp ipv4 unicast neighbors`

    ```bash
    myswitvh(config)# show bgp ipv4 unicast neighbors

        BGP neighbor is 10.10.12.5, remote AS 65000, ebgp link, Peer index 3
        BGP version 4, remote router ID 10.10.12.5
        BGP state = Established, up for 00:00:02
        Peer is directly attached, interface Ethernet1/1
        Last read 00:00:01, hold time = 30, keepalive interval is 10 seconds
        Last written 0.996717, keepalive timer expiry due 00:00:09
        Received 5 messages, 0 notifications, 0 bytes in queue
        Sent 13 messages, 0 notifications, 0(0) bytes in queue
        Connections established 1, dropped 0
        Last reset by us 00:01:29, due to session closed
        Last reset by peer never, due to No error

        Neighbor capabilities:
        Dynamic capability: advertised (mp, refresh, gr)
        Dynamic capability (old): advertised
        Route refresh capability (new): advertised received
        Route refresh capability (old): advertised
        4-Byte AS capability: advertised received
        Address family IPv4 Unicast: advertised received
        Graceful Restart capability: advertised
    ```

## 部署示例
1.  添加一个EIP池（单个eip请添加/32）
    ```bash
    kubectl apply -f - <<EOF
    apiVersion: network.kubesphere.io/v1alpha1
    kind: Eip
    metadata:
        name: eip-sample-pool
    spec:
        address: 10.11.11.0/24
        disable: false
    EOF
    ```

2. 部署测试Service. Service必须要添加如下一个annotations，type也要指定为LoadBalancer,如下：

    ```yaml
    kind: Service
    apiVersion: v1
    metadata:
        name:  mylbapp
        annotations:
            lb.kubesphere.io/v1alpha1: porter
            #eip.porter.kubesphere.io/v1alpha1: 1.1.1.1 如果需要手动指定eip，可以添加这个注记
    spec:
        selector:
            app:  mylbapp
        type:  LoadBalancer 
        ports:
        - name:  http
        port:  8088
        targetPort:  80
    ```

    可以使用我们提供的样例[Service](https://github.com/kubesphere/porter/blob/master/test/samples/eip.yaml),这个样例已经包含两个EIP，使用之前请确认与局域网不冲突

    ```bash
    kubectl apply -f service.yaml
    ``` 

3. 在路由器上查看是否有对应的路由。如果有，那么连接这个路由器的任何主机应该都能通过EIP+ServicePort的方式访问了。

    ```
    show routing
    ……
    10.11.11.11/32, ubest/mbest: 3/0
    *via 10.10.12.2, [20/0], 00:03:38, bgp-65001, external, tag 65000
    *via 10.10.12.3, [20/0], 00:03:38, bgp-65001, external, tag 65000
    *via 10.10.12.4, [20/0], 00:03:38, bgp-65001, external, tag 65000

    ```
4. 在集群上使用`kubectl get eip`观察当前集群eip的使用情况