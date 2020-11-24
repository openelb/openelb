# Porter layer2 mode

> English | [中文](zh/layer2.md)


Most routers now support BGP, but in practice it may be inconvenient to open up BGP for some reason, such as security compliance, or if the physical router you are using is too old to support BGP, then you can configure Porter layer2 mode to achieve similar functionality.

## Layer2 Principle

When a client accesses the server via IP, since the configured Eip is on the same Layer 2 network as the Kubernetes cluster, the router will send an ARP/NDP request to find the MAC address of the Eip. At this time Porter will answer the MAC of the Kubernetes Node according to the Endpoints of the LoadBalancer Service. After ARP/NDP is answered, subsequent client traffic is sent to the same Node.

Due to the one-to-one correspondence between IP and MAC, the LoadBalancer Service can only answer the MAC address of the same Node during its lifetime, unless Endpoints change. To achieve this, Porter uses Kubernetes' own Leader Election feature, which allows only one copy to answer ARP/NDP requests.

**Limitations: There is a single point of failure when the client connects to the server via Eip, and all traffic from Eip is sent to the same Node**.

Translated with www.DeepL.com/Translator (free version)


## The use of layer2

The layer2 mode is much simpler to use than the BGP mode. You just need to configure the layer2 mode Eip, and specify the protocol as layer2 when creating the workload.

* Create layer2 Eip
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

* Create workloads and services

To use layer2 in the Service, we need to use "protocol.porter.kubesphere.io/v1alpha1: layer2" to specify the use of layer2.

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

## Verify layer2

**Note: To verify the layer2 mode, you need to operate on a separate node outside the Kubernetes cluster **.

* View Service LoadBalancer IP
```bash
root@node1:~# kubectl get svc mylbapp-svc-layer2
NAME                   TYPE           CLUSTER-IP    EXTERNAL-IP    PORT(S)        AGE
mylbapp-svc-layer2   LoadBalancer   10.233.44.8   172.22.0.188   80:30564/TCP   10d
```

* View MAC address for Eip

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

The above operation reveals that the next hop for 172.22.0.188 is 172.22.0.3, since they both have the same MAC address and point to the same node, node1.

