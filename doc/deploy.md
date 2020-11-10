# Installation

> English | [中文](zh/deploy.md)

## Prerequisites

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