# porter chart

# 安装porter chart

```bash 
helm repo add test https://charts.kubesphere.io/test
help repo update
helm install porter test/porter
```

# layer2模式

## 前提条件

- Kubernetes集群，版本1.17.3及以上

- 局域网内一台linux机器hostA，用于检测nginx的LoadBalancer

## 配置layer2

```bash 
$ cat << EOF > layer2.yaml
apiVersion: network.kubesphere.io/v1alpha1
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    # 修改ip地址段为实际环境的ip地址段。可以为单个地址或者是地址段
    address: 192.168.3.100
    protocol: layer2
    disable: false
EOF
$ kubectl apply -f layer2.yaml
eip.network.kubesphere.io/eip-sample-pool created
```

## 部署nginx

在Kubernetes集群上:

```bash
$ cat << EOF > nginx-layer2.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: layer2
  name: nginx-service
spec:
  selector:
    app: nginx
  type:  LoadBalancer 
  ports:
    - name: http
      port: 8088
      targetPort: 80
EOF
$ kubectl apply -f nginx-layer2.yaml
deployment.apps/nginx-deployment created
service/nginx-service created
$ kubectl get svc/nginx-service
default       kubernetes      ClusterIP      10.96.0.1     <none>          443/TCP                  129m
default       nginx-service   LoadBalancer   10.100.5.90   192.168.3.100   8088:32063/TCP           50s
```

## 访问nginx服务

在hostA访问nginx

```bash
$ curl 192.168.3.100:8088
```


# BGP 模式

## 前提条件

- Kubernetes集群，版本1.17.3及以上。


- 开启BGP的路由器。在这里我们将在Centos7系统上安装bird，使用bird实现BGP路由功能。我们以router称这台机器。

- 局域网内一台linux机器hostA，用于检测nginx的LoadBalancer

## 网络图

```bash
 ________________             ________________              ________________
|               |            |                |            |                | 
| k8s cluster   | <--------- |     router     | <--------- |   other host   |
|_______________|            |________________|            |________________|
```

- router在这里是一个路由器，实验中我们没有具有bgp功能的路由器，因此使用一台主机替代。

- 其他主机将包发送个router，router在将包发送给k8s cluster。


- k8s cluster需要使用BGP协议和router建立连接，因此两者的as域必须不一样。

## router配置

在router上安装bird

```bash
$ yum install bird 
$ systemctl enable bird
```


在router上配置BGP，如下

```bash
cat /etc/bird.conf
protocol kernel {
    scan time 60;       # Scan kernel routing table every 20 seconds
    import none;        # Default is import all
    export all;         # Default is export none
    merge paths on;     # Enable ECMP, this parameter requires at least bird 1.6
}

protocol device {
    scan time 10;       # Scan interfaces every 10 seconds
}

protocol static {
}

protocol bgp mymaster {   
    description "10.55.0.127";                  # 本机ip地址
    local as 65001;                             # as域，必须和port-manager的as域不一样
    neighbor 10.55.0.124 port 17900 as 65000;   # port-manager的as域不一样
    source address 10.55.0.127;                 # 本机ip地址
    import all; 
    export all;
    enable route refresh off;
    add paths on;
}
```

在router上启动bird，并设置ipv4转发。

```bash
$ systemctl restart bird
$ sysctl -w net.ipv4.ip_forward=1
```

在router上查看配置是否生效,你会看到新添一条mymaster规则。

```bash
$ birdc show protocol
BIRD 1.6.8 ready.
name     proto    table    state  since       info
kernel1  Kernel   master   up     18:01:55    
device1  Device   master   up     18:01:55    
static1  Static   master   up     18:01:55    
mymaster BGP      master   start  18:01:55    Active        Socket: Connection refused
```



## 在porter和router上建立BGP连接


在Kubernetes上:

```bash 
$ cat << EOF > bgp.yaml
apiVersion: network.kubesphere.io/v1alpha1
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    # 修改ip地址段为实际环境的ip地址段。
    address: 10.55.0.100
    protocol: bgp
    disable: false
---
apiVersion: network.kubesphere.io/v1alpha1
kind: BgpConf
metadata:
  name: bgpconf-sample
spec:
  # 设置porter的as域
  as : 65000
  routerID : 10.55.0.124
  port: 17900
---
apiVersion: network.kubesphere.io/v1alpha1
kind: BgpPeer
metadata:
  name: bgppeer-sample
spec:
  # 设置需要建立连接的as域，这里使用route的as域
  config:
    peerAs : 65001
    neighborAddress: 10.55.0.127
  addPaths:
    sendMax: 10
EOF
$ kubectl apply -f bgp.yaml
eip.network.kubesphere.io/eip-sample-pool created
bgpconf.network.kubesphere.io/bgpconf-sample created
bgppeer.network.kubesphere.io/bgppeer-sample created
```

在router上查看是否建立连接,info信息显示Established表示建立连接。

```bash
$ birdc show protocol
BIRD 1.6.8 ready.
name     proto    table    state  since       info
kernel1  Kernel   master   up     18:10:39    
device1  Device   master   up     18:10:39    
static1  Static   master   up     18:10:39    
mymaster BGP      master   up     18:15:45    Established
```

## 部署nginx

```bash 
$ cat << EOF > nginx-bgp.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: bgp
  name: nginx-service
spec:
  selector:
    app: nginx
  type:  LoadBalancer 
  ports:
    - name: http
      port: 8088
      targetPort: 80
EOF
$ kubectl apply -f nginx-bgp.yaml
deployment.apps/nginx-deployment created
service/nginx-service created
```

## 访问nginx服务

如果局域网内其他机器想要访问nginx，需要在设置路由。将包转发给router。

```bash
$ #-host指单台机器，如果需要指定网段请使用-net
$ #192.168.3.100指应用服务的地址，这里使用nginx service地址
$ #192.168.3.85指route地址。
$ #eth0指网卡
$ route add -host 192.168.3.100 gw 192.168.3.85 eth0
```

```bash
$ curl 192.168.3.100:8088
```
