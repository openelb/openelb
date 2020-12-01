# Installation

> English | [中文](zh/deploy.md)

## Prerequisites

* Kubernetes Version >= 1.15

Porter uses the CRD resource version v1, which is only supported since kubernetes 1.15.

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

## Installation via kubectl

To install Porter in one click, run the following command

```bash
kubectl apply -f https://raw.githubusercontent.com/kubesphere/porter/master/deploy/porter.yaml
```

## Installation via chart package

```bash 
helm repo add test https://charts.kubesphere.io/test
helm repo update
helm install porter test/porter
```

## Installation on KubeSphere

* Importing the chart repo where the porter is located in the workspace
![image](https://user-images.githubusercontent.com/3678855/100723369-a486b980-33fc-11eb-90bd-9768ec26ebd3.png)

* In the project, select Create Application and choose Create from Template, select the repository you imported in the previous step, and choose porter

![image](https://user-images.githubusercontent.com/3678855/100723664-03e4c980-33fd-11eb-9ffb-7d1488705f3f.png)

![image](https://user-images.githubusercontent.com/3678855/100723740-1f4fd480-33fd-11eb-9fae-07e4be5b1474.png)

*  Click on the porter, and follow the wizard. Finally, modify the chart configuration according to your own configuration, and deploy it in the project.
![image](https://user-images.githubusercontent.com/3678855/100723851-3a224900-33fd-11eb-8d7d-152137e19936.png)

![image](https://user-images.githubusercontent.com/3678855/100723964-532afa00-33fd-11eb-9dcb-d2684f482dd0.png)



