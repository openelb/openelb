# 模拟路由器
> 本文是在[青云平台](https://www.qingcloud.com/)上做的实验，不同的平台有一些配置有区别。模拟的好处是能够不接触实际硬件的情况下体验Porter的功能，但和实际路由器还是有差别，模拟路由器有个默认的网卡（不用于路由），会导致报文回去的时候走这个默认路由。另外在配置模拟路由器时，有很多额外的细节参数，请按照本文所述的参数进行设置。

## 前提
1. 已经拥有一个正常运行的k8s集群
2. 确保用于模拟路由器的主机和k8s集群相互连通，包括集群中的bgp端口，和应用使用的端口。
3. 有可供实验的公网ip


## 创建路由器

1. 在k8s所在的网络内创建一个主机，最小配置即可。进入主机安装bird，青云平台主机上默认只有bird 1.5，1.5不支持ECMP，要体验Porter的全部功能需要至少1.6:
    ```bash
     $sudo add-apt-repository ppa:cz.nic-labs/bird 
     $sudo apt-get update 
     $sudo apt-get install bird
     $sudo systemctl enable bird  
    ```
2. 配置路由器的BGP服务。修改`/etc/bird/bird.conf`，添加如下参数：
    ```
    protocol bgp mymaster {   
        description "192.168.1.4"; ##填本交换机ID，一般是主IP   
        local as 65001; ##填本地AS域，必须和k8s集群的AS不同   
        neighbor 192.168.1.5 port 17900 as 65000; ##填master节点IP和 AS域   
        source address 192.168.1.4; ##填本交换机IP    
        import all;   
        export all;
        enable route refresh off; #由于bird1.6的BGP协议较低，和Porter的bgp连接会将多路由变成单个路由，这个参数能够作为一个workaround修正这个问题。
        add paths on; #这个参数开启之后，就可以收到porter发来的多个路由并同时存在而不会覆盖。
    }
     ```
   上述参数给模拟路由器配置了一个邻居，邻居即是集群主节点，**这里假设了porter的controller部署在了主节点，如果不想限制porter部署在主节点，或者master不能部署pod，那么在这里需要按照上述配置，将所有可能的邻居节点按照上述规则添加到这个配置文件中**。修改该文件中的kernel部分，将其中的export all的注释取消，并开启ECMP功能。修改为：
   ```
   protocol kernel {
        scan time 60;
        import none;
        export all;   # Actually insert routes into the kernel routing table
        merge paths on; #开启ECMP功能，这个参数至少需要 bird 1.6
   }

   ```

3. 重启bird
   ```bash
    $sudo systemctl restart bird 
   ```

4. 配置公网ip。在青云控制台上申请一个**内部绑定**的公网IP，注意必须是一个内部绑定的IP，如果是外部绑定的话是无法感知这个公网ip的。将这个ip绑定到模拟路由器所在主机上。绑定完成即可，不需要按照青云文档进行后续的操作。

5. 观察网卡。青云平台内部绑定的IP绑定到主机上时，会在主机上创建一个新的网卡（一般是eth1）,登录主机执行`ip a`查看这个网卡是否启动（状态是否为`UP`），如果没有启动，执行`ip link set up eth1`。eth1是这个ip的入口网卡。

6. 打开模拟路由器的端口转发并关闭包过滤  
    ```
    sysctl -w net.ipv4.ip_forward=1   
    sysctl -w net.ipv4.conf.all.rp_filter=0    
    sysctl -w net.ipv4.conf.eth1.rp_filter=0
    sysctl -w net.ipv4.conf.eth0.rp_filter=0
    ```
7. 配置防火墙。在青云控制台上打开一些测试端口，如8000-30000等。

8. 配置路由回路。由于模拟路由器的默认网卡是eth0，在集群返回ip包之后，默认会从eth0发出，而用户访问这个公网ip是从eth1进来的，这样就会导致信息发送失败，所以需要将从绑定的IP发来的包导流到eth1。在浏览器上访问这个地址，同时在模拟路由器上使用`tcpdump -i eth1`抓包，观察上层路由地址，如：

    ```bash
    root@i-7bwamgny:~# tcpdump -i eth1
    tcpdump: verbose output suppressed, use -v or -vv for full protocol decode
    listening on eth1, link-type EN10MB (Ethernet), capture size 262144 bytes
    14:24:07.401555 IP 139.198.254.4.1395 > 139.198.121.228.omniorb: Flags [S], seq 3677905607, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532475097 ecr 0], length 0
    14:24:07.403573 IP 139.198.254.4.1396 > 139.198.121.228.omniorb: Flags [S], seq 2462558694, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532475100 ecr 0], length 0
    14:24:07.654341 IP 139.198.254.4.1397 > 139.198.121.228.omniorb: Flags [S], seq 1471601642, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532475350 ecr 0], length 0
    14:24:10.400770 IP 139.198.254.4.1395 > 139.198.121.228.omniorb: Flags [S], seq 3677905607, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532478097 ecr 0], length 0
    14:24:10.404100 IP 139.198.254.4.1396 > 139.198.121.228.omniorb: Flags [S], seq 2462558694, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532478100 ecr 0], length 0
    14:24:10.658557 IP 139.198.254.4.1397 > 139.198.121.228.omniorb: Flags [S], seq 1471601642, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532478351 ecr 0], length 0
    14:24:16.401591 IP 139.198.254.4.1395 > 139.198.121.228.omniorb: Flags [S], seq 3677905607, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532484098 ecr 0], length 0
    14:24:16.404605 IP 139.198.254.4.1396 > 139.198.121.228.omniorb: Flags [S], seq 2462558694, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532484101 ecr 0], length 0
    14:24:16.656750 IP 139.198.254.4.1397 > 139.198.121.228.omniorb: Flags [S], seq 1471601642, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532484351 ecr 0], length 0

    ```
    上述打印输出中，`139.198.121.228`是绑定的ip，左边即上层路由器的地址。获取到这个地址之后，通过路由策略配置回去的规则：
    ```bash
    sudo ip rule add to 139.198.254.0/24 lookup 101 #返回这个ip的包走路由表101
    sudo ip route replace default via dev eth1 table 101 #路由表101的默认网卡是eth1
    ```
    实际物理路由器不需要配置上述规则，因为路由器知道如何正确配置这个ip。**如果需要从多个ip地址访问测试ECMP，那么这些IP也需要相同的步骤**

9. 这样模拟路由器就配置完成了，可以执行`birdc show protocol`查看连接信息。

## 配置插件
> 所有的操作都在k8s集群的主节点中

1. 获取yaml文件
    ```
    wget https://github.com/kubesphere/porter/releases/download/v0.0.1/porter.yaml
    ```
2. 修改yaml文件中的configmap `bgp-cfg`，请按照<https://github.com/kubesphere/porter/blob/master/doc/bgp_config.md>配置这个文件，并且需要和刚才模拟器配置相对应。
3. 配置公网ip回路规则。和模拟路由器的问题一致，公网ip导流至集群中之后，ip包发出默认都是eth0，eth0会将此包丢弃，需要将此ip包导向模拟路由器。**这一步需要在k8s所有节点上配置，因为实际的服务可能部署在任何节点。**
    ```bash
    sudo ip rule add to 139.198.254.0/24 lookup 101 #返回这个ip的包走路由表101
    sudo ip route replace default via 192.168.98.5 dev eth0 table 101 #路由表101的默认网关是192.168.98.5这个模拟路由器
    ```
    上面的`192.168.98.5`即模拟路由器的地址，模拟路由器上已经配置了一条回路规则，所以此包就不会被丢弃了。实际k8s集群不需要配置，因为k8s集群的默认网关就是这个路由器。
4. （可选，必读）配置master节点label，我们需要强制将porter部署到master节点。
   ```bash
   kubectl label nodes name_of_your_master dedicated=master #请先修改mastername
   ```
   如果不想限制porter部署的节点，那么需要在上面配置交换机的时候，将所有节点都作为模拟路由器的邻居。并且删除porter.yaml中的nodeselector:
   ```yaml
    nodeSelector:
        dedicated: master
    ##如果不想限制porter控制器部署的节点，那么需要注释掉上面两行
   ```
5. 安装porter到集群中，`kubectl apply -f release.yaml`
6. 添加一个EIP到集群中。
   ```bash
   kubectl apply -f - <<EOF
    apiVersion: network.kubesphere.io/v1alpha1
    kind: EIP
    metadata:
    labels:
        controller-tools.k8s.io: "1.0"
    name: eip-sample
    spec:
    # Add fields here
        address: 139.198.121.228 #这里替换为你申请的EIP
        disable: false
    EOF 
   ```
7. 部署测试Service. Service必须要添加如下一个annotations，type也要指定为LoadBalancer,如下：

    ```yaml
    kind: Service
    apiVersion: v1
    metadata:
    name:  mylbapp
    annotations:
        lb.kubesphere.io/v1alpha1: porter
    spec:
        selector:
            app:  mylbapp
        type:  LoadBalancer 
        ports:
        - name:  http
            port:  8088
            targetPort:  80
    ```
    可以使用我们提供的样例[Service](https://github.com/kubesphere/porter/blob/master/config/sample/service.yaml)
    > 使用这个样例之前需先替换里面的EIP
    ```
    kubectl apply -f service.yaml
    ``` 
8. 检查一下Porter日志和EIP events，如果没问题，就可以按照Service中的EIP和其端口访问服务了。
   ```bash
   kubectl logs -n porter-system controller-manager-0 -c manager
   kubectl describe eip eip-sample ##观察是否有对应的event
   ```
9. 检查模拟路由器上，是否有两个等价路由：
   ```bash
   root@i-7bwamgny:~# ip route
   default via 192.168.98.1 dev eth0
   139.198.121.228  proto bird
        nexthop via 192.168.98.2  dev eth0 weight 1
        nexthop via 192.168.98.4  dev eth0 weight 1
   ```

## 测试ECMP带来的Load Balancing 功能
> 注，模拟路由器所在的主机内核版本要高于3.6，青云平台默认内核为4.4，使用的ECMP Hash算法是`L3`的，ECMP只会根据源IP调整访问的路由。4.12以上的内核支持设置`L4`的。，可以设置`sysctl net.ipv4.fib_multipath_hash_policy 1`改变负载均衡hash算法，那么利用curl访问这个eip也可以实现测试负载均衡的效果。

实际路由器只需要开启ECMP功能就可以实现负载均衡，为了测试负载均衡有效，需要从不同的源IP访问这个EIP，同时在Pod分布的节点上用tcpdump观察是否有流量。
1. 首先观察Pod所在节点
```bash
kubectl get pod -o wide
```
2. 观察各个节点是否设置了引流规则
```bash
root@master-k8s:~# ip rule
0:      from all lookup local
32763:  from all to 139.198.121.228 lookup 101
```
3. 然后在这些节点上运行tcpdump -i eth0 port $port, $port就是服务对外暴露的端口，上面的例子中就是8088。
4. 从不同的IP访问这个EIP，观察在这些节点上是否有对应的流量，如果都有，那么负载均衡就没有问题。

