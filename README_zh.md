# OpenELB 

![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/kubesphere/openelb) ![go report](https://goreportcard.com/badge/github.com/kubesphere/openelb)

![OpenELB Logo](./doc/logo/color-horizontal.svg)

OpenELB是一个开源的负载均衡器实现,可以在裸金属服务器、边缘以及虚拟化环境中使用LoadBalancer类型的service.目前作为[沙箱项目](https://www.cncf.io/sandbox-projects/)托管于[CNCF](https://www.cncf.io/).

## 为什么选择OpenELB

在云服务环境中的kubernetes集群里,通常可以用云服务提供商提供的负载均衡服务来暴露service,但是在本地没办法这样操作.而OpenELB可以让用户在裸金属服务器、边缘以及虚拟化环境中创建LoadBalancer类型的service来暴露服务,并且可以做到和云环境中的用户体验是一致的.

## 核心功能

- BGP模式和二层网络模式下的负载均衡
- ECMP路由和负载均衡
- IP池管理
- 基于CRD来管理BGP配置
- 支持Helm Chart方式安装

## 快速入门

- [在Kubernetes中安装OpenELB](https://openelb.github.io/docs/getting-started/installation/install-porter-on-kubernetes/)
- [在K3s中安装OpenELB](https://openelb.github.io/docs/getting-started/installation/install-porter-on-k3s/)
- [在KubeSphere中安装OpenELB](https://openelb.github.io/docs/getting-started/installation/install-porter-on-kubesphere/)

## 文档

你可以按照[OpenELB文档](https://openelb.github.io/docs/)中的步骤来学习如何在云服务K8S中部署OpenELB.

## 采用者

OpenELB已经被采用在[很多公司](./ADOPTERS.md),如果你也正在使用OpenELB,欢迎加入到用户社区并且把你的logo添加到[采用者列表](./ADOPTERS.md)!

## 开发计划

[OpenELB开发计划](doc/roadmap.md)列出了每个里程碑下的功能以及bug修复,如果你有新的想法、功能需求或者建议,欢迎提交proposal.

## 参与贡献以及讨论

* 加入[KubeSphere Slack Channel](https://kubesphere.slack.com/join/shared_invite/enQtNTE3MDIxNzUxNzQ0LTZkNTdkYWNiYTVkMTM5ZThhODY1MjAyZmVlYWEwZmQ3ODQ1NmM1MGVkNWEzZTRhNzk0MzM5MmY4NDc3ZWVhMjE#/)来咨询问题或告诉我们你正在使用OpenELB.(很快将会有kubernetes下的slack channel)
* 欢迎任何文档完善以及代码贡献!具体可以看[贡献指南](https://openelb.github.io/docs/building-and-contributing/).

## License

OpenELB is licensed under the Apache License, Version 2.0. See [LICENSE](./LICENSE) for the full license text.
