# Porter layer2模式

现在大多数路由器都支持BGP，但是在实际应用过程中或多或少会由于某些原因而不便开放BGP功能， 例如安全合规， 或者用户使用的物理路由器实在太老不支持BGP， 那么这里可以通过配置Porter layer2模式用以达到类似的功能。

## layer2 原理

当客户端通过IP访问服务端时，由于配置的Eip与Kubernetes集群处于同一二层网络， 路由器会通过发送ARP/NDP请求查找Eip对应的MAC地址，这个时候Porter会根据LoadBalancer Service的Endpoints应答Kubernetes Node对应MAC。ARP/NDP应答完成之后，后续客户端流量都会发往同一个Node。

由于IP与MAC的一一对应关系， 在LoadBalancer Service的生命周期内只能应答同一个Node的MAC地址，除非Endpoints变化。 为了做到这一点， Porter采用Kubernetes自带的Leader Election功能， 通过它实现只会有一个副本应答ARP/NDP请求。

**限制：客户端通过Eip连接服务端时会存在单点故障， Eip的所有流量都会发往同一个Node**


## layer2的使用

在使用上， layer2模式较BGP模式简单很多，只需要配置layer2模式的Eip，并在创建工作负载时指定protocol为layer2

* 创建layer2 eip
```yaml
kubectl apply -f - <<EOF
apiVersion: network.kubesphere.io/v1alpha2
kind: Eip
metadata:
  name: eip-sample-layer2
spec:
  address: 172.22.0.188-172.22.0.200
  interface: eth0
  protocol: layer2
EOF
```

* 创建工作负载以及服务

在Service中使用layer2的时候，我们需要使用"protocol.porter.kubesphere.io/v1alpha1: layer2"指定使用layer2.

```yaml
kubectl apply -f - <<EOF
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc-layer2
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: layer2
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

## 验证layer2

**为了验证layer2模式， 你需要在Kubernetes集群外一台单独节点操作以下步骤**

* 查看Service LoadBalancer IP
```bash
root@node1:~# kubectl get svc mylbapp-svc-layer2
NAME                   TYPE           CLUSTER-IP    EXTERNAL-IP    PORT(S)        AGE
mylbapp-svc-layer2   LoadBalancer   10.233.44.8   172.22.0.188   80:30564/TCP   10d
```

* 查看Eip对应MAC地址

```bash
root@i-7iisycou:~# ping 172.22.0.188
PING 172.22.0.188 (172.22.0.188) 56(84) bytes of data.
64 bytes from 172.22.0.188: icmp_seq=1 ttl=64 time=14.7 ms
64 bytes from 172.22.0.188: icmp_seq=2 ttl=64 time=1.04 ms
^C
--- 172.22.0.188 ping statistics ---
2 packets transmitted, 2 received, 0% packet loss, time 1001ms
rtt min/avg/max/mdev = 1.048/7.911/14.775/6.864 ms
root@i-7iisycou:~# ip neigh
172.22.0.188 dev eth0 lladdr 52:54:22:40:2a:66 DELAY
172.22.0.3 dev eth0 lladdr 52:54:22:40:2a:66 DELAY
```

```bash
root@node1:~# kubectl get nodes -o wide
NAME    STATUS   ROLES           AGE   VERSION   INTERNAL-IP   EXTERNAL-IP   OS-IMAGE             KERNEL-VERSION       CONTAINER-RUNTIME
node1   Ready    master,worker   18d   v1.17.9   172.22.0.3    <none>        Ubuntu 18.04.4 LTS   4.15.0-109-generic   docker://19.3.6
node3   Ready    worker          18d   v1.17.9   172.22.0.9    <none>        Ubuntu 18.04.4 LTS   4.15.0-108-generic   docker://19.3.8
node4   Ready    worker          18d   v1.17.9   172.22.0.10   <none>        Ubuntu 18.04.4 LTS   4.15.0-101-generic   docker://19.3.8
root@node1:~#
```

通过以上操作发现172.22.0.188下一跳即为172.22.0.3， 因为他们的MAC地址都相同，指向同一节点node1。


