# Porter Chart

> English | [中文](zh/porter-chart.md)

# Install Porter using Helm Chart

```bash 
helm repo add test https://charts.kubesphere.io/test
help repo update
helm install porter test/porter
```

# Layer 2 mode

## Prerequistes

- Requires Kubernetes `1.17.3` or above

- A linux machine, used to detect LoadBalancer of nginx

## Configure layer2 in kubernetes

```bash 
$ cat << EOF > layer2.yaml
apiVersion: network.kubesphere.io/v1alpha1
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    # Modify the ip address segment to the ip address segment of the actual environment. It can be a single address or an address segment
    address: 192.168.3.100
    protocol: layer2
    disable: false
EOF
$ kubectl apply -f layer2.yaml
eip.network.kubesphere.io/eip-sample-pool created
```

## Deploy nginx

Execute commands on Kubernetes cluster:

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

## Visit nginx service

Execute commands on linux:

```bash
$ curl 192.168.3.100:8088
```

# BGP mode

## Prerequistes

- Requires Kubernetes `1.17.3` or above.

- Require `router` to start BGP mode.We will install bird on Centos7 system and use bird to implement BGP routing function. We call this machine `router`.

- A linux machine, used to detect LoadBalancer of nginx

## Network diagram

```bash
 ________________             ________________              ________________
|               |            |                |            |                | 
| k8s cluster   | <--------- |     router     | <--------- |   other host   |
|_______________|            |________________|            |________________|
```

- We use bird to implement BGP mode on centos7 system.

- Other hosts send packets to a `router`, and the `router` is sending packets to the k8s cluster.

- The k8s cluster needs to use BGP to establish a connection with the `router`, so the `as`  of the two must be different.

## configure on router

Install bird on `router`:

```bash
$ yum install bird 
$ systemctl enable bird
```

Configure BGP on the `router` as follows:

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
    description "10.55.0.127";                  # local ip
    local as 65001;                             # local as.It must be different from the as of the port-manager
    neighbor 10.55.0.124 port 17900 as 65000;   # Master node IP and AS number
    source address 10.55.0.127;                 # Router IP 
    import all; 
    export all;
    enable route refresh off;
    add paths on;
}
```

Start bird on the `router` and set ipv4 forwarding:

```bash
$ systemctl restart bird
$ sysctl -w net.ipv4.ip_forward=1
```

Check whether the configuration takes effect on the `router`, you will see `mymaster` rule.

```bash
$ birdc show protocol
BIRD 1.6.8 ready.
name     proto    table    state  since       info
kernel1  Kernel   master   up     18:01:55    
device1  Device   master   up     18:01:55    
static1  Static   master   up     18:01:55    
mymaster BGP      master   start  18:01:55    Active        Socket: Connection refused
```


## Establish BGP connection on porter and router

Execute commands on Kubernetes cluster:

```bash 
$ cat << EOF > bgp.yaml
apiVersion: network.kubesphere.io/v1alpha1
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    # Modify the ip address segment to the ip address segment of the actual environment.
    address: 10.55.0.100
    protocol: bgp
    disable: false
---
apiVersion: network.kubesphere.io/v1alpha1
kind: BgpConf
metadata:
  name: bgpconf-sample
spec:
  # the as of porter
  as : 65000
  routerID : 10.55.0.124
  port: 17900
---
apiVersion: network.kubesphere.io/v1alpha1
kind: BgpPeer
metadata:
  name: bgppeer-sample
spec:
  # the as of the router
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

Check whether the connection is established on the `router`, and the info information shows `Established` means the connection is established.

```bash
$ birdc show protocol
BIRD 1.6.8 ready.
name     proto    table    state  since       info
kernel1  Kernel   master   up     18:10:39    
device1  Device   master   up     18:10:39    
static1  Static   master   up     18:10:39    
mymaster BGP      master   up     18:15:45    Established
```

## Deploy nginx

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

## Visit nginx service

If other machines in the LAN want to access nginx, you need to set up routing. Forward the packet to the `router`.

```bash
$ # "-host" refers to a single machine, if you need to specify a network segment, please use "-net"
$ #"192.168.3.100" refers to the address of the application service.This use the nginx service address
$ #"192.168.3.85" refers to the router address.
$ route add -host 192.168.3.100 gw 192.168.3.85 eth0
```

```bash
$ curl 192.168.3.100:8088
```
