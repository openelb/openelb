![GitHub version](https://img.shields.io/badge/version-v0.0.1-brightgreen.svg?logo=appveyor&longCache=true&style=flat)
![go report](https://goreportcard.com/badge/github.com/kubesphere/porter)

# Porter

> English | [中文](README_zh.md)

`Porter` is a load balancer which is designed for a bare metal Kubernetes cluster. It's implemented by physical switch, and uses BGP and ECMP to achieve optimal performance and high availability. As we know, In the cloud-enabled Kubernetes cluster, the cloud provider usually provides the LB plugin to assign EIP and expose the service externally. However, the service is  hard to expose externally in a bare metal cluster because of lacking a cloud Load Balancer. In short, Porter allows you to create Kubernetes services of type “LoadBalancer” in bare metal clusters, which makes you have consistency experiences on the cloud. This plugin has two components which provide following core features: 

1. LB Controller & Agent: The controller is responsible for synchronizing BGP routes to the physical switch; The agent is deployed to each node as DaemonSet to maintain the drainage rules;

2. The EIP service, including the EIP pools management and EIP controller, is responsible for dynamically updating the EIP information of the service.

> Note: Porter is a subproject of [KubeSphere](https://github.com/kubesphere/kubesphere).


## Principle

The following figure desribes the principle of Porter. Suppose there is a distributed service on node1 (192.168.0.2) and node2 (192.168.0.6). The service needs to be accessed through EIP `1.1.1.1`. After deploying the [Example Service](https://github.com/kubesphere/porter/blob/master/test/samples/test.yaml), Porter will automatically synchronize routing information to the leaf switch, and then synchronize to the spine and border switch, thus external users can access the service through EIP `1.1.1.1`.

![node architecture](doc/img/node-arch.png)

## Deployment Architecture

The plugin monitors the changes of the service in the cluster through a `Manager` and broadcasts related routes. At the same time, all the nodes in the cluster are deployed with an agent. Whenever an EIP is used, a host routing rule will be added to the host, diverting the IP packets sent to the EIP to the local device.

![porter deployment](doc/img/porter-deployment.png)

## Logic

When Porter is deployed as a service in Kubernetes cluster, it establishes a BGP connection with the cluster's border router (Layer 3 switch). When a service with a specific annotation (an annotation is `lb.kubesphere.io/v1apha1: porter`, see [Example Service](https://github.com/kubesphere/porter/blob/master/config/samples/service.yaml)) has been created in the cluster, the service is dynamically assigned EIP (user can also specify EIP by itself). The LB controller creates a route and forwards the route to the public network (or private network) through BGP, so that the service can be accessed externally.

The Porter LB controller is a custom controller based on the [Kubernetes controller runtime](https://github.com/kubernetes-sigs/controller-runtime) that automatically changes routing information by watching changes of the service.

![porter architecture](doc/img/porter-arch.png)

## Installation

1. [Deploy Porter on Bare Metal Kubernetes Cluster](doc/deploy_baremetal.md)
2. [Test Porter on QingCloud with a Simulate Router](doc/simulate_with_bird.md)

## Build 

### Prerequisites

1. Go 1.11, the plugin uses [gobgp](https://github.com/osrg/gobgp) to create BGP client, and godgp requires Go 1.11.
2. Docker
3. Kustomize，it uses [kustomize](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/INSTALL.md) to dynamically generate the k8s yaml files needed for the cluster.
4. If you need to push the plugin image to the remote private repository, you need to execute `docker login` in advance.

### Steps

1. Execute `git clone https://github.com/kubesphere/porter.git`, then enter into the folder.
2. Following with the above guides to modify the config.toml (Under `config/bgp/`). 
3. (Optional）Modify the code according to your needs.
4. (Optional）Modify the parameters of the image according to your needs (Under `config/manager`).
5. (Optional）Follow the [Simulation Tutorial](doc/simulate_with_bird.md) to deploy a Bird node, then modify the BirdIP in `hack/test.sh`, and run `make e2e-test` for e2e testing.
6. Modify the IMG name in the Makefile, then run `make release`, and the final yaml file is under `/deploy`.
7. Execute `kubectl apply -f deploy/release.yaml` to deploy porter as a plugin.

## Licence

**Porter** is licensed under the Apache License, Version 2.0. See [LICENSE](./LICENSE) for the full license text.



