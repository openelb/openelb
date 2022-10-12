<p align="center">
<a href="https://openelb.github.io/"><img src="docs/logo/openelb-vertical.svg" alt="banner" width="70px"></a>
</p>

<p align="center">
<b>Load Balancer Implementation for <i>Kubernetes in Bare-Metal, Edge, and Virtualization</i></b>
</p>

<p align=center>
<a href="https://goreportcard.com/report/github.com/openelb/openelb"><img src="https://goreportcard.com/badge/github.com/openelb/openelb" alt="A+"></a>
<a href="https://hub.docker.com/r/kubesphere/openelb"><img src="https://img.shields.io/docker/pulls/kubesphere/openelb"></a>
<a href="https://github.com/openelb/openelb/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22"><img src="https://img.shields.io/github/issues/openelb/openelb/good%20first%20issue.svg" alt="good first"></a>
<a href="https://twitter.com/intent/follow?screen_name=KubeSphere"><img src="https://img.shields.io/twitter/follow/KubeSphere?style=social" alt="follow on Twitter"></a>
<a href="https://join.slack.com/t/kubesphere/shared_invite/enQtNTE3MDIxNzUxNzQ0LTZkNTdkYWNiYTVkMTM5ZThhODY1MjAyZmVlYWEwZmQ3ODQ1NmM1MGVkNWEzZTRhNzk0MzM5MmY4NDc3ZWVhMjE"><img src="https://img.shields.io/badge/Slack-600%2B-blueviolet?logo=slack&amp;logoColor=white"></a>
<a href="https://www.youtube.com/channel/UCyTdUQUYjf7XLjxECx63Hpw"><img src="https://img.shields.io/youtube/channel/subscribers/UCyTdUQUYjf7XLjxECx63Hpw?style=social"></a>
</p>

## OpenELB：云原生负载均衡器插件

_用其他语言阅读: [English](README.md), [中文](README_zh.md)._

![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/kubesphere/openelb) ![go report](https://goreportcard.com/badge/github.com/kubesphere/openelb)

OpenELB 是一个开源的云原生负载均衡器实现，可以在基于裸金属服务器、边缘以及虚拟化的 Kubernetes 环境中使用 LoadBalancer 类型的 Service 对外暴露服务。

OpenELB 项目最初由 [KubeSphere 社区](https://kubesphere.io) 发起，目前已作为 CNCF [沙箱项目](https://www.cncf.io/sandbox-projects/) 加入 CNCF 基金会，由 OpenELB 开源社区维护与支持。

## 为什么选择 OpenELB

在云服务环境中的 Kubernetes 集群里，通常可以用云服务提供商提供的负载均衡服务来暴露 Service，但是在本地没办法这样操作。而 OpenELB 可以让用户在裸金属服务器、边缘以及虚拟化环境中创建 LoadBalancer 类型的 Service 来暴露服务，并且可以做到和云环境中的用户体验是一致的。

## 核心功能

- BGP 模式和二层网络模式下的负载均衡
- ECMP 路由和负载均衡
- IP 池管理
- 基于 CRD 来管理 BGP 配置
- 支持 Helm Chart 方式安装

## 快速入门

- [在 Kubernetes 中安装 OpenELB](https://openelb.github.io/docs/getting-started/installation/install-openelb-on-kubernetes/)
- [在 K3s 中安装 OpenELB](https://openelb.github.io/docs/getting-started/installation/install-openelb-on-k3s/)
- [在 KubeSphere 中安装 OpenELB](https://openelb.github.io/docs/getting-started/installation/install-openelb-on-kubesphere/)

## 文档

您可以按照[OpenELB 文档](https://openelb.github.io/docs/)中的步骤来学习如何在云服务 K8S 中部署 OpenELB。

## 采用者

OpenELB 已经被采用在[很多公司](./ADOPTERS.md)，如果您也正在使用 OpenELB，欢迎加入到用户社区并且把您所在组织或企业的 Logo 添加到[采用者列表](./ADOPTERS.md)！

## 开发计划

[OpenELB 开发计划](docs/roadmap.md)列出了每个里程碑下的功能以及 Bug 修复。如果您有新的想法、功能需求或者建议,欢迎提交 proposal。

## 参与贡献以及讨论

- 加入 [Slack Channel](https://kubesphere.slack.com/join/shared_invite/enQtNTE3MDIxNzUxNzQ0LTZkNTdkYWNiYTVkMTM5ZThhODY1MjAyZmVlYWEwZmQ3ODQ1NmM1MGVkNWEzZTRhNzk0MzM5MmY4NDc3ZWVhMjE#/)来咨询问题或告诉我们您正在使用 OpenELB（很快将会有 Kubernetes 下的 Slack Channel）
- 欢迎任何文档完善以及代码贡献!具体可以看[贡献指南](https://openelb.github.io/docs/building-and-contributing/)

## License

OpenELB 采用 Apache 2.0 开源协议，详见 [LICENSE 源文件](./LICENSE)。
