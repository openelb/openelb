# 模拟路由器

> [English](../simulate_with_bird.md) | 中文

作为普通开发者，日常工作中很难接触到硬件路由器，幸运地是很多软件可以提供物理路由器类似的功能，例如[bird](https://bird.network.cz/)、[gobgp](https://osrg.github.io/gobgp/)，它们同样能够协助我们开发。

## 前提

1. 拥有一个正常运行的Kubernetes集群
2. Porter已正确安装
3. 用于模拟路由器的主机和Kubernetes集群网络互通

## 创建并配置路由器

1. 安装bird。 
    
`1.5不支持ECMP，要体验Porter的全部功能需要至少1.6`, 在ubuntu中可以执行下面的脚本安装bird 1.6
    
```
$sudo add-apt-repository ppa:cz.nic-labs/bird ##这里引入了bird 1.6
$sudo apt-get update 
$sudo apt-get install bird
$sudo systemctl enable bird  
```
   
2. 配置bird

bird的配置文件为`/etc/bird/bird.conf`, 具体配置可以参考[bird官方文档](https://bird.network.cz/?get_doc&f=bird.html&v=16)
  
* 配置router id

`router id`格式为合法的ip地址， 请根据实际环境修改   

```
router id 172.22.0.2;
```

* 配置bgp neighbor

根据`Porter Manager`实际部署节点配置bgp neighbor， **如有多个请对应添加多个neighbor**。

```
protocol bgp neighbor1 {   
    local as 65001; #填本地AS域，必须和Kubernetes集群的AS不同   
    neighbor 172.22.03 port 17900 as 65000; ##填master节点IP和 AS域   
    source address 172.22.0.2; #填本交换机IP    
    import all;   
    export all;
    enable route refresh off; #由于bird1.6的bgp较低，和Porter的bgp连接会将多路由变成单个路由，这个参数能够作为一个workaround修正这个问题。
    add paths on; #这个参数开启之后，就可以收到porter发来的多个路由并同时存在而不会覆盖。
}
```
  
* 配置bird kernel
   
添加`export all;` 和 `merge paths on;` 用于向linux kernel添加路由。
   
```
protocol kernel {
    scan time 60;
    import none;
    export all;
    merge paths on; #开启ECMP功能，这个参数至少需要 bird 1.6
}
```

3. 重启bird

```bash
$sudo systemctl restart bird 
```

4. 确认bird

通过以下命令查看bird配置是否正常启动。如果状态为非`active`, 可以执行`journalctl -f -u  bird`查看错误。

```bash
$sudo systemctl status bird 
```

## 配置Porter

* 配置BgpConf

请参考[bgp_config](./bgp_config.md)， 按照自己需求修改下面配置

```yaml
kubectl apply -f - <<EOF
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
EOF
```

* 配置BgpPeer

请参考[bgp_config](./bgp_config.md)， 按照自己需求修改下面配置

```yaml
kubectl apply -f - <<EOF
apiVersion: network.kubesphere.io/v1alpha2
kind: BgpPeer
metadata:
  name: bgppeer-sample
spec:
  conf:
    peerAs: 50000
    neighborAddress: 172.22.0.2
EOF
```

* 配置Eip

请参考[eip_config](./eip_config.md)， 按照自己需求修改下面配置

```yaml
kubectl apply -f - <<EOF
apiVersion: network.kubesphere.io/v1alpha2
kind: Eip
metadata:
  name: eip-sample
spec:
  address: 139.198.121.228
EOF
```

## 部署测试工作负载与服务

执行如下命令创建工作负载以及LoadBalancer Service

```yaml
kubectl apply -f - <<EOF
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: bgp
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  ports:
    - name:  http
      port:  8088
      targetPort:  80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: mylbapp
  name: mylbapp
spec:
  replicas: 2
  selector:
    matchLabels:
      app: mylbapp
  template:
    metadata:
      labels:
        app: mylbapp
    spec:
      containers:
        - image: nginx:alpine
          name: nginx
          ports:
            - containerPort: 80
EOF
```

Service必须要添加annotations `lb.kubesphere.io/v1alpha1: porter`，type也要指定为`LoadBalancer`

## 检查路由

执行以下命令，检查模拟路由器上是否有等价路由：
```bash
root@i-7iisycou:/tmp# ip route
139.198.121.228 proto bird metric 64
        nexthop via 172.22.0.3 dev eth0 weight 1
        nexthop via 172.22.0.9 dev eth0 weight 1
        nexthop via 172.22.0.10 dev eth0 weight 1
```