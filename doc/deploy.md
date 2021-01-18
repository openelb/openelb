# Installation

> English | [中文](zh/deploy.md)

## Prerequisites

* Kubernetes Version >= 1.15

Porter uses the CRD resource version v1, which is only supported since kubernetes 1.15.

* Set node label 
  
When the cluster nodes are distributed under different tor switches, you need to label the node to identify the network topology
```bash
kubectl label --overwrite nodes i-9reu0ohi  porter.kubesphere.io/rack=leaf3
```
Then label the nodes that need to deploy the porter-manager. Ensure that at least two porter-managers exist under at least one tor switch in a production environment
```bash
kubectl label nodes i-y0a0i550 lb.kubesphere.io/v1alpha1=porter
```

Modify porter-manager deployment's nodeselector
```yaml
nodeSelector:
  kubernetes.io/os: linux
  lb.kubesphere.io/v1alpha1: porter
```
```bash
kubectl rollout restart -n porter-system deployment porter-manager
```

* BGP mode

1. The router must support the BGP protocol.
2. Requires the router to support ECMP and includes the following features.
    - Support for receiving multiple equivalence routes
    - Supports receiving multiple equivalent routes from the same neighbor.
    
* layer2 mode

In layer2 mode, you need to enable `strictARP`, which disables the network card from answering ARP requests from IP addresses on other network cards.

```yaml
kubectl edit configmap -n kube-system kube-proxy

apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: "ipvs"
ipvs:
  strictARP: true
```

Then restart the kube-proxy

```bash
kubectl rollout restart -n kube-system daemonset kube-proxy
```

Note: When there are multiple NICs in the cluster node and the eip you need to configure for layer2 mode is not in node.Status.Addresses, you need to manually set annotation `layer2.porter.kubesphere.io/v1alpha1` for each node as the ip on the corresponding NIC address

If two NICs exist in your node, configure the IPs separately as follows.
1. eth0 172.28.0.2/24  (Internal use)
2. eth1 172.30.0.2/24  (External use)

At this time you install the node status field as follows
```yaml
status:
  addresses:
  - address: 172.28.3.4
    type: InternalIP
  - address: i-y0a0i550
    type: Hostname
```
In order to access the eip in 172.30.0.0/24 properly, you need to set the annotation in nodes
```bash
kubectl annotate nodes i-9reu0ohi  layer2.porter.kubesphere.io/v1alpha1="172.30.0.2"
```

## Three ways to install porter

### Installation via kubectl

To install Porter in one click, run the following command

```bash
kubectl apply -f https://raw.githubusercontent.com/kubesphere/porter/master/deploy/porter.yaml
```

### Installation via chart package

```bash 
helm repo add test https://charts.kubesphere.io/test
helm repo update
helm install porter test/porter
```

### Installation on KubeSphere

* Importing the chart repo where the porter is located in the workspace
![image](https://user-images.githubusercontent.com/3678855/100723369-a486b980-33fc-11eb-90bd-9768ec26ebd3.png)

* In the project, select Create Application and choose Create from Template, select the repository you imported in the previous step, and choose porter

![image](https://user-images.githubusercontent.com/3678855/100723664-03e4c980-33fd-11eb-9ffb-7d1488705f3f.png)

![image](https://user-images.githubusercontent.com/3678855/100723740-1f4fd480-33fd-11eb-9fae-07e4be5b1474.png)

*  Click on the porter, and follow the wizard. Finally, modify the chart configuration according to your own configuration, and deploy it in the project.
![image](https://user-images.githubusercontent.com/3678855/100723851-3a224900-33fd-11eb-8d7d-152137e19936.png)

![image](https://user-images.githubusercontent.com/3678855/100723964-532afa00-33fd-11eb-9dcb-d2684f482dd0.png)
