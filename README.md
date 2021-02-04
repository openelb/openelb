![GitHub version](https://img.shields.io/badge/version-v0.0.1-brightgreen.svg?logo=appveyor&longCache=true&style=flat)
![go report](https://goreportcard.com/badge/github.com/kubesphere/porter)

# Porter: Load Balancer Implementation for Bare Metal

![logo](doc/img/porter-logo.png)

[Porter](https://porter.kubesphere.io/) is an open-source load balancer implementation designed for bare-metal Kubernetes clusters.

## Why Porter

In cloud-based Kubernetes clusters, services are usually exposed by using load balancers provided by cloud vendors. However, cloud-based load balancers are unavailable in bare-metal environments. Porter allows users to create LoadBalancer services in bare-metal environments for external access, and provides the same user experience as cloud-based load balancers.

## Core Features

- BGP mode and Layer 2 mode
- ECMP routing and load balancing
- IP address pool management
- BGP configuration using CRDs
- Installation using Helm and KubeSphere

## Documentation

Without a bare-metal environment yet? Doesn't matter!

You can learn how to use Porter in a cloud-based Kubernetes cluster by following the [Porter Documentation](./doc/index.md).

## Support, Discussion and Contributing

Porter is a sub-project of [KubeSphere](https://github.com/kubesphere/kubesphere).

* Join us at the [KubeSphere Slack Channel](https://kubesphere.slack.com/join/shared_invite/enQtNTE3MDIxNzUxNzQ0LTZkNTdkYWNiYTVkMTM5ZThhODY1MjAyZmVlYWEwZmQ3ODQ1NmM1MGVkNWEzZTRhNzk0MzM5MmY4NDc3ZWVhMjE#/) to get support or simply tell us that you are using Porter.
* You have code or documents for Porter? We ❤️ all sorts of contributions! You can [build the Porter project](/doc/how-to-build.md) and send us pull requests.

## Landscapes

<p align="center">
<br/><br/>
<img src="https://landscape.cncf.io/images/left-logo.svg" width="150"/>&nbsp;&nbsp;<img src="https://landscape.cncf.io/images/right-logo.svg" width="200"/>&nbsp;&nbsp;
<br/><br/>
Porter is a promising newcomer in service proxy, which enriches the <a href="https://landscape.cncf.io/landscape=observability-and-analysis&license=apache-license-2-0">CNCF CLOUD NATIVE Landscape.
</a>
</p>

## License

**Porter** is licensed under the Apache License, Version 2.0. See [LICENSE](./LICENSE) for the full license text.