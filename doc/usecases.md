# Layer2

[Layer2 guide](layer2.md) demonstrates how to install Porter and use Layer 2 on Kubernetes.

## Prerequisite

The cluster should have at least two nodes, nodeA and nodeB.

## Test endpoint update

- Create a Service test-svc which has one endpoint on nodeA.
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: mylbapp
  labels:
    app: mylbapp
spec:
  containers:
  - name: helloworld
    image: karthequian/helloworld
    imagePullPolicy: IfNotPresent
  nodeSelector:
    kubernetes.io/hostname: nodeA
---
kind: Service
apiVersion: v1
metadata:
    name:  test-svc
    annotations:
        lb.kubesphere.io/v1alpha1: porter
        #eip.porter.kubesphere.io/v1alpha1: 1.1.1.1 
        #protocol.porter.kubesphere.io/v1alpha1: layer2 
spec:
    selector:
        app:  mylbapp
    type:  LoadBalancer 
    ports:
      - name:  http
        port:  80
        targetPort:  80
```

- Execute 'wget ${eip}' can successfully access to test-svc
- Execute 'ip neigh  show to ${eip} dev ${nic}', we can find lladdr is nodeA's Mac
- Delete the endpoint which on nodeA, and create endpoint on nodeB
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: mylbapp
  labels:
    app: mylbapp
spec:
  containers:
  - name: helloworld
    image: karthequian/helloworld
    imagePullPolicy: IfNotPresent
  nodeSelector:
    kubernetes.io/hostname: nodeB
```
- Execute 'wget ${eip}' can successfully access to test-svc
- Execute 'ip neigh show to ${eip} dev ${nic}', we can find lladdr is nodeB's Mac

# BGP

[BGP guide](simulate_with_bird.md) demonstrates how to install Porter and use BGP on Kubernetes.

## Prerequisite

The cluster should have at least two nodes, nodeA and nodeB.

## Test BGP PortForward
- Configure BgpConf with port 17900
```yaml
apiVersion: network.kubesphere.io/v1alpha1
kind: BgpConf
metadata:
  name: bgpconf-sample
spec:
  # Add fields here
  as : 65000
  routerID : 192.168.0.2
  port: 17900
```
- Configure BgpPeer with usingPortForward
```yaml
apiVersion: network.kubesphere.io/v1alpha1
kind: BgpPeer
metadata:
  name: bgppeer-sample
spec:
  # Add fields here
  usingPortForward: true
  config:
    peerAs : 65001
    neighborAddress: 192.168.0.6
  addAaths:
    sendMax: 10
```
- Configure bird  neighbour  with port 179
- Execute 'birdctl show protocol', the nighbour's state should be up

## Test BGP passive mode
- Configure BgpConf with port 17900
- Set passiveMode to true in BgpPeer configuration.
```yaml
apiVersion: network.kubesphere.io/v1alpha1
kind: BgpPeer
metadata:
  name: bgppeer-sample
spec:
  config:
    peerAs : 65001
    neighborAddress: 192.168.0.6
  addAaths:
    sendMax: 10
  transport:
    passiveMode: true
```
- Configure bird neighbour with port 17900
- Execute 'birdctl show protocol', the nighbour's state should be up

## Test endpoint update
- Create a Service test-svc which has two endpoints on nodeA and nodeB respectively.
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: mylbapp
  labels:
    app: mylbapp
spec:
  containers:
  - name: helloworld
    image: karthequian/helloworld
    imagePullPolicy: IfNotPresent
  nodeSelector:
    kubernetes.io/hostname: nodeA
---
apiVersion: v1
kind: Pod
metadata:
  name: mylbapp2
  labels:
    app: mylbapp
spec:
  containers:
  - name: helloworld
    image: karthequian/helloworld
    imagePullPolicy: IfNotPresent
  nodeSelector:
    kubernetes.io/hostname: nodeB
---
kind: Service
apiVersion: v1
metadata:
    name:  test-svc
    annotations:
        lb.kubesphere.io/v1alpha1: porter
        #eip.porter.kubesphere.io/v1alpha1: 1.1.1.1 
        #protocol.porter.kubesphere.io/v1alpha1: layer2 
spec:
    selector:
        app:  mylbapp
    type:  LoadBalancer 
    ports:
      - name:  http
        port:  80
        targetPort:  80
```
- Execute 'wget ${eip}' can successfully access to test-svc
- Execute 'ip route get ${eip}' on test node, we can find two routes, one via nodeA and one via nodeB.
- Delete the endpoint which on nodeA
- Execute 'wget ${eip}' can successfully access to test-svc
- Execute 'ip route get ${eip}' on test node, we can find one route via nodeA.

## Test BGP graceful down
- Create a Service test-svc which has one endpoint on nodeA
- Execute 'wget ${eip}' can successfully access to test-svc 
- Execute 'ip route get ${eip}' on test node, we can find one route via nodeA.
- Execute 'kubectl scale -n porter-system deployment porter-manager  --replicas=0'
- Execute 'wget ${eip}' can successfully access to test-svc 
- Execute 'ip route get ${eip}' on test node, we can find one route via nodeA.
- Execute 'kubectl scale -n porter-system deployment porter-manager  --replicas=1'
- Execute 'wget ${eip}' can successfully access to test-svc 
- Execute 'ip route get ${eip}' on test node, we can find one route via nodeA.
