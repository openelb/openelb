# Changelog
All notable changes to this project will be documented in this file.

## [ 0.6.0 ]

### Changes

* Split Manager into Speaker and Controller. [#332](https://github.com/openelb/openelb/pull/332)  [#335](https://github.com/openelb/openelb/pull/335)  [#338](https://github.com/openelb/openelb/pull/338) [#380](https://github.com/openelb/openelb/pull/380) [#402](https://github.com/openelb/openelb/pull/402) [#408](https://github.com/openelb/openelb/pull/408)
    * Controller: Focuses on IP allocation and reclamation for LoadBalancer type services.
    * Speaker: Focuses on announcing the allocated IPs. Now supports three modes: BGP, Layer 2, and VIP.

### New Features

* Eip supports binding to namespaces. [#352](https://github.com/openelb/openelb/pull/352)
* Eip supports priority configuration for IPAM. [#352](https://github.com/openelb/openelb/pull/352)
* Added dynamic enable switches for VIP and Layer 2 modes. 
* Added support for multiple network interfaces in VIP and Layer 2 modes. [#408](https://github.com/openelb/openelb/pull/408)

### Bug Fixes

* Introduced memberlist for node failure detection in Layer 2 mode to resolve single point of failure issues. [#380](https://github.com/openelb/openelb/pull/380)
* Fixed support for `externalTrafficPolicy` configuration in VIP mode. [#408](https://github.com/openelb/openelb/pull/408)
* Fixed data residue issues after uninstalling VIP mode.  [#420](https://github.com/openelb/openelb/pull/420)

### Optimizations

* Optimized the IPAM process in the controller.
* Reduced the number of required annotations for services.
* Improved compatibility with other CNI plugins.

## [ 0.5.1 ] - 2022-07-17

### New Feature
* Add uninstall script. [#276](https://github.com/openelb/openelb/pull/276)
* Add some metrics. [#293](https://github.com/openelb/openelb/pull/293)
* Support config for container image. [#285](https://github.com/openelb/openelb/pull/285)

### Enhancements
* Add github templates. [#279](https://github.com/openelb/openelb/pull/279)
* Fix wrong image on Makefile. [#284](https://github.com/openelb/openelb/pull/284)
* Minor changes to codebase. [#294](https://github.com/openelb/openelb/pull/294)

## [ 0.5.0 ] - 2022-05-20

### Feature
- Support vip mode. [#252](https://github.com/openelb/openelb/pull/252) 

### Upgrade
* Use constants of gobgp api. [#253](https://github.com/openelb/openelb/pull/253)
* Add all contributors. [#261](https://github.com/openelb/openelb/pull/261)

## [ 0.4.4 ] - 2022-02-26

### Cleanup
- Rename PorterLB to OpenELB. [#242](https://github.com/openelb/openelb/pull/242)

## [ 0.4.3 ] - 2022-01-29

### Feature
- Support assign EIP by default with a config to controller. [#236](https://github.com/openelb/openelb/pull/236)

### Upgrade
- Upgrade kube-webhook-certgen. [#234](https://github.com/openelb/openelb/pull/234)

## [ 0.4.2 ] - 2021-07-8

### Feature
- Using the CNI plugin as a speaker for synchronous routes. [#199](https://github.com/kubesphere/porterlb/pull/199)
- Rename PorterLB to OpenELB. [#207](https://github.com/kubesphere/openelb/pull/207)

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