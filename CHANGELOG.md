# Changelog
All notable changes to this project will be documented in this file.
## [ 0.0.3 ] - 2019-03-26

### Added
 - new Jenkinsfile <https://github.com/kubesphere/porter/pull/29>

### Fixed
 - duplicated externalIPs in `kubectl get svc` <https://github.com/kubesphere/porter/pull/27>
 - update docs

## [ 0.0.2 ] - 2019-03-25

### Added
 - auto detect main interface instead of using `eth0` <https://github.com/kubesphere/porter/pull/23>
 - add e2e test <https://github.com/kubesphere/porter/pull/23>

### Fixed
 - fix the reconcile logic which add route without waiting for all endpoints  <https://github.com/kubesphere/porter/pull/24>
 - Update readme