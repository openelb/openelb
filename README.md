![GitHub version](https://img.shields.io/badge/version-v0.0.1-brightgreen.svg?logo=appveyor&longCache=true&style=flat)
![go report](https://goreportcard.com/badge/github.com/kubesphere/openelb)

# OpenELB

OpenELB is an open-source load balancer implementation designed for exposing the LoadBlancer type of Kubernetes services in bare metal, edge, and virtualization environments.

## Why OpenELB

In cloud-based Kubernetes clusters, Services are usually exposed by using load balancers provided by cloud vendors. However, cloud-based load balancers are unavailable in bare-metal or on-premise environments. OpenELB allows users to create **LoadBalancer Services** in bare-metal, egde, and virtualization environments for external access, and provides the same user experience as cloud-based load balancers.

## Core Features

- Load balancing in BGP mode and Layer 2 mode
- ECMP routing and load balancing
- IP address pool management
- BGP configuration using CRDs
- Installation using Helm Chart

## Quickstart

- [Install OpenELB on Kubernetes](https://porterlb.io/docs/getting-started/installation/install-porter-on-kubernetes/)
- [Install OpenELB on K3s](https://porterlb.io/docs/getting-started/installation/install-porter-on-k3s/)
- [Install OpenELB on KubeSphere](https://porterlb.io/docs/getting-started/installation/install-porter-on-kubesphere/)

## Documentation

Without a bare-metal environment yet? Doesn't matter!

You can learn how to use OpenELB in a cloud-based Kubernetes cluster by following the [OpenELB Documentation](https://porterlb.io/docs/).

## Adopters

OpenELB has been adopted by [many companies](./ADOPTERS.md) all over the world. If you are using OpenELB in your organization, welcome to join the end user community and add your logo to the [list](./ADOPTERS.md)!

## Roadmap

[OpenELB Roadmap](doc/roadmap.md) lists the features and bug fixes for each milestone. If you have any new ideas, feature requests or suggestions, please submit a proposal. 

## Support, Discussion and Contributing

OpenELB is a sub-project of [KubeSphere](https://github.com/kubesphere/kubesphere).

* Join us at the [KubeSphere Slack Channel](https://kubesphere.slack.com/join/shared_invite/enQtNTE3MDIxNzUxNzQ0LTZkNTdkYWNiYTVkMTM5ZThhODY1MjAyZmVlYWEwZmQ3ODQ1NmM1MGVkNWEzZTRhNzk0MzM5MmY4NDc3ZWVhMjE#/) to get support or simply tell us that you are using OpenELB.
* You have code or documents for OpenELB? Contributions are always welcome! See [Building and Contributing](https://porterlb.io/docs/building-and-contributing/) to obtain guidance.

## Landscapes

<p align="center">
<br/><br/>
<img src="https://landscape.cncf.io/images/left-logo.svg" width="150"/>&nbsp;&nbsp;<img src="https://landscape.cncf.io/images/right-logo.svg" width="200"/>&nbsp;&nbsp;
<br/><br/>
OpenELB is a promising newcomer in Service proxy, which enriches the <a href="https://landscape.cncf.io/landscape=observability-and-analysis&license=apache-license-2-0">CNCF CLOUD NATIVE Landscape.
</a>
</p>


## License

OpenELB is licensed under the Apache License, Version 2.0. See [LICENSE](./LICENSE) for the full license text.
