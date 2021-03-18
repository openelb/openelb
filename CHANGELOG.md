# Changelog
All notable changes to this project will be documented in this file.

## [ 0.4.1 ] - 2021-03-18

### **BugFix:**
- Fix the name of the program in the image.[#196](https://github.com/kubesphere/porterlb/pull/196)

### **Misc Changes:**
- Change the `deploy/porter.yaml` image name to `kubesphere/porter:v0.4.1`

## [ 0.4.0 ] - 2021-03-16

### **Feature:**
- Eip Address Management via CRD.[#132](https://github.com/kubesphere/porter/pull/132)
- Changes to the BgpConf/BgpPeer API to be compatible with the gobgp API and to support viewing status. [#132](https://github.com/kubesphere/porter/pull/132)

### **BugFix:**
- Add param to config webhook port. [#136](https://github.com/kubesphere/porter/pull/136)
- Filter not ready nodes from nexthops. [#142](https://github.com/kubesphere/porter/pull/142)

## [ 0.3.1 ] - 2020-08-28

### **Feature:**
- Supports automatic builds using GitHub actions [#122](https://github.com/kubesphere/porter/pull/122) [#123](https://github.com/kubesphere/porter/pull/123)

## [ 0.3.0 ] - 2020-08-28

### **Feature:**
- Support layer 2 load-balancing
- Support loadBalancerIP in Service
- Support add neighbor dynamically
- Support config porter via CRD

### [ 0.1.1 ] - 2019-09-09

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