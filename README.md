![GitHub version](https://img.shields.io/badge/version-v0.0.1-brightgreen.svg?logo=appveyor&longCache=true&style=flat)
![go report](https://goreportcard.com/badge/github.com/kubesphere/porter)

# Porter

`Porter`是一款用于物理机部署kubernetes的服务暴露插件，是[Kubesphere](https://kubesphere.io/)的一个子项目。

## 工作原理

该插件部署在集群中时，会与集群的边界路由器（三层交换机）建立BGP连接。每当集群中创建了带有porter注记的服务时，就会为该服务动态分配EIP，EIP将以辅助IP的形式绑定在Controller所在的节点主网卡上，然后创建路由，BGP将路由传导到公网（私网）中，使得外部能够访问这个服务。

![architecture](https://github.com/kubesphere/porter/blob/master/doc/img/logic.png)

## 部署插件

1. [在物理部署的k8s集群上部署](https://github.com/kubesphere/porter/blob/master/doc/deploy_baremetal.md)
2. [在青云上用模拟路由器的方式开始](https://github.com/kubesphere/porter/blob/master/doc/simulate_with_bird.md)

## 服务的Porter注记
如果应用想要使用Porter来暴露服务，需要在应用对应的服务中创建如下标记：
```yaml
annotations:
    lb.kubesphere.io/v1alpha1: porter
```
同时，服务的类型需要为`LoadBalancer`
```yaml
spec:
    type: LoadBalancer
```
可以参考代码仓库中的[nginx样例](https://github.com/kubesphere/porter/blob/master/config/sample/service.yaml)

## 物理架构
![architecture](https://github.com/kubesphere/porter/blob/master/doc/img/architecture.png)

## 从代码构建新的插件

### 软件需求
1. go 1.11，插件使用了[gobgp](https://github.com/osrg/gobgp)创建BGP服务端，gobgp需要go 1.11
2. docker，无版本限制
3. kustomize，插件使用了[kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md)动态生成集群所需的k8s yaml文件
4. 如果插件会推送到远端私有仓库，需要提前执行`docker login`

### 步骤
1. `git clone https://github.com/kubesphere/porter.git`, 进入代码目录 
2. `dep ensure --vendor-only`拉取依赖
3. 按照上面教程的要求修改config.toml (位于`config/bgp/`下） 
4. （optional）根据自己需要修改代码
5. （optional）根据自己的需求修改镜像的参数（位于`config/manager`下）
6. 修改Makefile中的IMG名称，然后`make release`，最终的yaml文件在`deploy`目录下
7. `kubectl apply -f deploy/release.yaml` 部署插件

## 开源许可

**Porter** is licensed under the Apache License, Version 2.0. See
[LICENSE](https://github.com/kubesphere/porter/blob/master/LICENSE) for the full
license text.