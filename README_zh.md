# OpenELB 

![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/kubesphere/openelb) ![go report](https://goreportcard.com/badge/github.com/kubesphere/openelb)

![OpenELB Logo](./doc/logo/color-horizontal.svg)

OpenELB 是一个开源的负载均衡器实现,可以在裸金属服务器、边缘以及虚拟化环境中使用 LoadBalancer 类型的Service.目前作为[沙箱项目](https://www.cncf.io/sandbox-projects/)托管于[CNCF](https://www.cncf.io/).

## 为什么选择OpenELB

在云服务环境中的 Kubernetes 集群里,通常可以用云服务提供商提供的负载均衡服务来暴露 Service,但是在本地没办法这样操作.而 OpenELB 可以让用户在裸金属服务器、边缘以及虚拟化环境中创建 LoadBalancer 类型的 Service 来暴露服务,并且可以做到和云环境中的用户体验是一致的.

## 核心功能

- BGP 模式和二层网络模式下的负载均衡
- ECMP 路由和负载均衡
- IP 池管理
- 基于 CRD 来管理 BGP 配置
- 支持 Helm Chart 方式安装

## 快速入门

- [在 Kubernetes 中安装 OpenELB](https://openelb.github.io/docs/getting-started/installation/install-porter-on-kubernetes/)
- [在 K3s 中安装 OpenELB](https://openelb.github.io/docs/getting-started/installation/install-porter-on-k3s/)
- [在 KubeSphere 中安装 OpenELB](https://openelb.github.io/docs/getting-started/installation/install-porter-on-kubesphere/)

## 文档

您可以按照[OpenELB 文档](https://openelb.github.io/docs/)中的步骤来学习如何在云服务 K8S 中部署 OpenELB.

## 采用者

OpenELB 已经被采用在[很多公司](./ADOPTERS.md),如果您也正在使用 OpenELB,欢迎加入到用户社区并且把您的logo添加到[采用者列表](./ADOPTERS.md)!

## 开发计划

[OpenELB 开发计划](doc/roadmap.md)列出了每个里程碑下的功能以及bug修复,如果您有新的想法、功能需求或者建议,欢迎提交proposal.

## 参与贡献以及讨论

* 加入[KubeSphere Slack Channel](https://kubesphere.slack.com/join/shared_invite/enQtNTE3MDIxNzUxNzQ0LTZkNTdkYWNiYTVkMTM5ZThhODY1MjAyZmVlYWEwZmQ3ODQ1NmM1MGVkNWEzZTRhNzk0MzM5MmY4NDc3ZWVhMjE#/)来咨询问题或告诉我们您正在使用OpenELB.(很快将会有kubernetes下的slack channel)
* 欢迎任何文档完善以及代码贡献!具体可以看[贡献指南](https://openelb.github.io/docs/building-and-contributing/).

## License

OpenELB is licensed under the Apache License, Version 2.0. See [LICENSE](./LICENSE) for the full license text.
