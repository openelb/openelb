<p align="center">
<a href="https://openelb.github.io/"><img src="doc/logo/openelb-vertical.svg" alt="banner" width="70px"></a>
</p>

<p align="center">
<b>Load Balancer Implementation for <i>Kubernetes in Bare-Metal, Edge, and Virtualization</i></b>
</p>

<p align=center>
<a href="https://goreportcard.com/report/github.com/openelb/openelb"><img src="https://goreportcard.com/badge/github.com/openelb/openelb" alt="A+"></a>
<a href="https://hub.docker.com/r/kubesphere/openelb"><img src="https://img.shields.io/docker/pulls/kubesphere/openelb"></a>
<a href="https://github.com/openelb/openelb/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22"><img src="https://img.shields.io/github/issues/badges/shields/good%20first%20issue" alt="good first"></a>
<a href="https://twitter.com/intent/follow?screen_name=KubeSphere"><img src="https://img.shields.io/twitter/follow/KubeSphere?style=social" alt="follow on Twitter"></a>
<a href="https://join.slack.com/t/kubesphere/shared_invite/enQtNTE3MDIxNzUxNzQ0LTZkNTdkYWNiYTVkMTM5ZThhODY1MjAyZmVlYWEwZmQ3ODQ1NmM1MGVkNWEzZTRhNzk0MzM5MmY4NDc3ZWVhMjE"><img src="https://img.shields.io/badge/Slack-600%2B-blueviolet?logo=slack&amp;logoColor=white"></a>
<a href="https://www.youtube.com/channel/UCyTdUQUYjf7XLjxECx63Hpw"><img src="https://img.shields.io/youtube/channel/subscribers/UCyTdUQUYjf7XLjxECx63Hpw?style=social"></a>
</p>

## OpenELB: Cloud Native Load Balancer Implementation

> English | [中文](README_zh.md)

OpenELB is an open-source load balancer implementation designed for exposing the LoadBalancer type of Kubernetes services in bare metal, edge, and virtualization environments. 

OpenELB was originally created by [KubeSphere](https://kubesphere.io) and is currently a vendor neutral and CNCF [Sandbox Project](https://www.cncf.io/sandbox-projects/).

## Why OpenELB

In cloud-based Kubernetes clusters, Services are usually exposed by using load balancers provided by cloud vendors. However, cloud-based load balancers are unavailable in bare-metal or on-premise environments. OpenELB allows users to create **LoadBalancer Services** in bare-metal, egde, and virtualization environments for external access, and provides the same user experience as cloud-based load balancers.

## Core Features

- Load balancing in BGP mode and Layer 2 mode
- ECMP routing and load balancing
- IP address pool management
- BGP configuration using CRDs
- Installation using Helm Chart

## Quickstart

- [Install OpenELB on Kubernetes](https://openelb.github.io/docs/getting-started/installation/install-openelb-on-kubernetes/)
- [Install OpenELB on K3s](https://openelb.github.io/docs/getting-started/installation/install-openelb-on-k3s/)
- [Install OpenELB on KubeSphere](https://openelb.github.io/docs/getting-started/installation/install-openelb-on-kubesphere/)
## Documentation

You can learn how to use OpenELB in a cloud-based Kubernetes cluster by following the [OpenELB Documentation](https://openelb.github.io/docs/).

## Adopters

OpenELB has been adopted by [many companies](./ADOPTERS.md) all over the world. If you are using OpenELB in your organization, welcome to join the end user community and add your logo to the [list](./ADOPTERS.md)!

## Roadmap

[OpenELB Roadmap](doc/roadmap.md) lists the features and bug fixes for each milestone. If you have any new ideas, feature requests or suggestions, please submit a proposal. 

## Support, Discussion and Contributing

* Join us at the [KubeSphere Slack Channel](https://kubesphere.slack.com/join/shared_invite/enQtNTE3MDIxNzUxNzQ0LTZkNTdkYWNiYTVkMTM5ZThhODY1MjAyZmVlYWEwZmQ3ODQ1NmM1MGVkNWEzZTRhNzk0MzM5MmY4NDc3ZWVhMjE#/) to get support or simply tell us that you are using OpenELB.(openelb slack channel from kubernetes will be soon)
* You have code or documents for OpenELB? Contributions are always welcome! See [Building and Contributing](https://openelb.github.io/docs/building-and-contributing/) to obtain guidance.

## License

OpenELB is licensed under the Apache License, Version 2.0. See [LICENSE](./LICENSE) for the full license text.
