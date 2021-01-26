# Install Porter on Kubernetes (kubectl and Helm)

This document describes how to use kubectl and Helm to install Porter in a Kubernetes cluster. For details about how to install Porter on the [KubeSphere](https://kubesphere.io/docs/installing-on-linux/introduction/multioverview/#step-3-create-a-cluster) web console, see [Install Porter on KubeSphere (Web Console)](./install-porter-on-kubesphere.md).

## Prerequisites

* You need to prepare a Kubernetes cluster, and ensure that the Kubernetes version is 1.15 or later. Porter requires CustomResourceDefinition (CRD) v1, which is only supported by Kubernetes 1.15 or later. You can use the following methods to deploy a Kubernetes cluster:

  * Use [KubeKey](https://kubesphere.io/docs/installing-on-linux/) (recommended). You can use KubeKey to deploy a Kubernetes cluster with or without KubeSphere.
  * Follow [official Kubernetes guides](https://kubernetes.io/docs/home/).

  Porter is designed to be used in bare-metal Kubernetes environments. However, you can also use a cloud-based Kubernetes cluster for learning and testing.

* If you use Helm to install porter, ensure that the Helm version is Helm 3.

## Install Porter Using kubectl

Log in to the Kubernetes cluster over SSH and run the following command:

```bash
kubectl apply -f https://raw.githubusercontent.com/kubesphere/porter/master/deploy/porter.yaml
```

## Install Porter Using Helm

Log in to the Kubernetes cluster over SSH and run the following commands:

```bash 
helm repo add test https://charts.kubesphere.io/test
helm repo update
helm install porter test/porter
```
