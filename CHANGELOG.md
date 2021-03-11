# Changelog
All notable changes to this project will be documented in this file.
## [ 0.1.1 ] - 2019-09-09

### Added
- [🚒 use annotation to store eip](https://github.com/kubesphere/porterlb/pull/57)
- [add Porter intro of English version](https://github.com/kubesphere/porterlb/pull/53)

### Changed
- [🌟 upgrade to kubebuilder 2.0](https://github.com/kubesphere/porterlb/pull/54)
- [⏫upgrade kustomize](https://github.com/kubesphere/porterlb/pull/55)

### Fixed
- [🚒 fix e2e](https://github.com/kubesphere/porterlb/pull/56)

## [ 0.1.0 ] - 2019-03-31

### Added
 - add portforward for nonstandard bgp port <https://github.com/kubesphere/porterlb/pull/37>
 - add doc about setting up in real router <https://github.com/kubesphere/porterlb/pull/36>
 - more tests


## [ 0.0.3 ] - 2019-03-26

### Added
 - new Jenkinsfile <https://github.com/kubesphere/porterlb/pull/29>

### Fixed
 - duplicated externalIPs in `kubectl get svc` <https://github.com/kubesphere/porterlb/pull/27>
 - update docs

## [ 0.0.2 ] - 2019-03-25

### Added
 - auto detect main interface instead of using `eth0` <https://github.com/kubesphere/porterlb/pull/23>
 - add e2e test <https://github.com/kubesphere/porterlb/pull/23>

### Fixed
 - fix the reconcile logic which add route without waiting for all endpoints  <https://github.com/kubesphere/porterlb/pull/24>
 - Update readme