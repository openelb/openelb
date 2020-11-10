# Simulate with bird

> English | [中文](zh/simulate_with_bird.md)

As an ordinary developer, it is difficult to come across hardware routers in your daily work. Fortunately, there are many software programs that provide similar functionality to physical routers, such as [bird](https://bird.network.cz/) and [gobgp](https://osrg.github.io/gobgp/), which can also assist with We develop.

## Prerequisites

1. have a functioning Kubernetes cluster
2. Porter is correctly installed
3. hosts with bird installed interoperate with the Kubernetes node network.

## Create and configure bird

1. Install bird 
    
`1.5 does not support ECMP, you need at least 1.6 to experience the full functionality of Porter`, in ubuntu you can install bird 1.6 by executing the following script    

```
$sudo add-apt-repository ppa:cz.nic-labs/bird ##这里引入了bird 1.6
$sudo apt-get update 
$sudo apt-get install bird
$sudo systemctl enable bird  
```
   
2. Configuring the bird

The configuration file of bird is `/etc/bird/bird.conf`, please refer to [bird official documentation](https://bird.network.cz/?get_doc&f=bird.html&v=16).
  
* Configuring the router id

The format of `router id` is a valid IP address, please change it according to your environment.  

```
router id 172.22.0.2;
```

* Configuring the bgp neighbor

Configure the bgp neighbor according to the actual deployment node of `Porter Manager`, **If you have multiple nodes, please add multiple neighbors**.

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
  
* Configuring the bird kernel
   
Add `export all;` and `merge paths on;` for adding routes to the linux kernel.
   
```
protocol kernel {
    scan time 60;
    import none;
    export all;
    merge paths on;
}
```

3. Restart bird

```bash
$sudo systemctl restart bird 
```

4. Confirm Bird

Check to see if the bird configuration is starting properly by running the following command. If the status is not `active`, you can check for errors by running `journalctl -f -u bird`.

```bash
$sudo systemctl status bird 
```

## Configuring Porter

* Configuring BgpConf

Please refer to [bgp_config] (. /bgp_config.md), and modify the following configuration as you see fit.

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

* Configuring BgpPeer

Please refer to [bgp_config] (. /bgp_config.md), and modify the following configuration as you see fit.

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

* Configuring Eip

Please refer to [eip_config] (.eip_config.md) and modify the following configuration according to your needs. /eip_config.md), and modify the following configuration as you see fit.

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

## Deploy test workloads and services

Execute the following commands to create the workload and LoadBalancer Service.

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

The service must add annotations `lb.kubesphere.io/v1alpha1: porter` and the type must be specified as `LoadBalancer`.

## Checking the routing table

Execute the following command to check for equivalent routes on the host with bird installed.

```bash
root@i-7iisycou:/tmp# ip route
139.198.121.228 proto bird metric 64
        nexthop via 172.22.0.3 dev eth0 weight 1
        nexthop via 172.22.0.9 dev eth0 weight 1
        nexthop via 172.22.0.10 dev eth0 weight 1
```