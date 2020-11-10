# 安装部署

> [English](../deploy.md) | 中文

## 安装前提

* BGP模式

1. 路由器必须支持BGP协议
2. 需要路由器支持ECMP，并包括以下特性：
    - 支持接收多个等价路由
    - 支持接收来自同一个邻居的多条等价路由
    
* layer2模式

在layer2模式下，需要开启`strictARP`, 禁止网卡应答其他网卡上IP地址地ARP请求

```yaml
kubectl edit configmap -n kube-system kube-proxy

apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: "ipvs"
ipvs:
  strictARP: true
```

然后重启kube-proxy
```bash
kubectl rollout restart -n kube-system daemonset kube-proxy
```

## 通过kubectl安装

执行以下命令即可一键安装Porter

```bash
kubectl apply -f https://raw.githubusercontent.com/kubesphere/porter/master/deploy/porter.yaml
```

## 通过chart包安装

```bash 
helm repo add test https://charts.kubesphere.io/test
helm repo update
helm install porter test/porter
```