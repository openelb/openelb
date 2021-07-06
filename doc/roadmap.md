PorterLB Roadmap lists the features and Bugfix for each milestone.

# Release Goals

| Edition  | Schedule |
|---|---|
| Release v0.6.0| Nov, 2021 |
| Release v0.5.0| Aug, 2021 |
| Release v0.4.0| Mar, 2021 |
| Release v0.3.1| Aug, 2020 |
| Release v0.3.0| Aug, 2020 |
| Release v0.1.0| Mar, 2019 |
| Release v0.0.1| Feb, 2019 |

# v0.6.0

- [] Achieve the High Availability in VIP mode based on Keepalived
- [] Achieve the LoadBalancer for kube-apiserver
- [] Support for BGP policies
- [] Support VIP Group and more VIPs
- [] Support for IPv6

# v0.5.0

- [] Provide the PorterLB Web UI for managing EIP and IP pool
- [] Support for more routing protocols; Be compatible with the popular CNI
- [] Provide the Prometheus metrics for monitoring
- [] Integration with KubeSphere Web Console [#1449](https://github.com/kubesphere/console/pull/1449)

# v0.4.0
## Feature
- [x] Eip Address Management via CRD.[#132](https://github.com/kubesphere/porter/pull/132)
- [x] Changes to the BgpConf/BgpPeer API to be compatible with the gobgp API and to support viewing status. [#132](https://github.com/kubesphere/porter/pull/132)

## BugFix
- [x] Add param to config webhook port. [#136](https://github.com/kubesphere/porter/pull/136)
- [x] Filter not ready nodes from nexthops. [#142](https://github.com/kubesphere/porter/pull/142)

# v0.3.1
## Feature
- [x] Supports automatic builds using GitHub actions [#122](https://github.com/kubesphere/porter/pull/122) [#123](https://github.com/kubesphere/porter/pull/123)

# v0.3.0
## Feature
- [x] Support layer 2 load-balancing
- [x] Support loadBalancerIP in Service
- [x] Support add neighbor dynamically
- [x] Support config porter via CRD

# v0.1.0
## Feature
- [x] add portforward for nonstandard bgp port
- [x] add doc about setting up in real router
- [x] more tests

# v0.0.1
## Feature
- [x] Release Porter v0.0.1