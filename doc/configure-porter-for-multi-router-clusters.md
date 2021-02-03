# Configure Porter for Multi-router Clusters (BGP Mode)

This document describes how to configure Porter in BGP mode for Kubernetes cluster nodes deployed under multiple routers. You can skip this document if all Kubernetes cluster nodes are deployed under the same router.

{{< notice note >}}

This document applies only to the BGP mode. The Layer 2 mode requires that all Kubernetes cluster nodes be on the same Layer 2 network (under the same router).

{{</ notice >}}

## Network Topology Before Configuration

This section explains why you need to perform the configuration. The following figure shows the network topology of a Kubernetes cluster before the configuration.

![multi-router-topology-1](./img/configure-porter-for-multi-router-clusters/multi-router-topology-1.jpg)

IP addresses in the preceding figure are examples only. The topology is described as follows:

* In the Kubernetes cluster, the master and worker 1 nodes are deployed under the leaf 1 BGP router, and the worker 2 node is deployed under the leaf 2 BGP router. Porter is only installed under leaf 1 (by default, only one Porter replica is installed).
* A service backed by two pods is deployed in the Kubernetes cluster, and is assigned an IP address 172.22.0.2 for external access. Pod 1 and pod 2 are deployed on worker 1 and worker 2 respectively.
* Porter establishes a BGP connection with leaf 1 and publishes the IP addresses of the master node and worker 1 (192.168.0.3 and 192.168.0.4) to leaf 1 as the next hop destined for the service IP address 172.22.0.2.
*  Leaf 1 establishes a BGP connection with the spine BGP router and publishes its own IP address 192.168.0.2 to the spine router as the next hop destined for the service IP address 172.22.0.2.
* When an external client machine attempts to access the service, the spine router forwards the service traffic to leaf 1, and leaf 1 load balances the traffic among the master node and worker 1.
* Although pod 2 on worker 2 can also be reached over kube-proxy, router-level load balancing is implemented only among the master node and worker 1 and the service bandwidth is limited to the bandwidth of the master node and worker 1.

To resolve the problem, you need to label the Kubernetes cluster nodes and change the Porter deployment configuration so that Porter is installed on nodes under all leaf routers. In addition, you need to specify the [spec.nodeSelector.matchLabels](./configure-porter-in-bgp-mode.md/#configure-peer-bgp-properties-using-bgppeer) field in the BgpPeer configuration so that the Porter replicas establish BGP connections with the correct BGP routers.

## Network Topology After Configuration

This section describes the configuration result you need to achieve. The following figure shows the network topology of a Kubernetes cluster after the configuration.

![multi-router-topology-2](./img/configure-porter-for-multi-router-clusters/multi-router-topology-2.jpg)

IP addresses in the preceding figure are examples only. The topology is described as follows:

* After the configuration, Porter is installed on nodes under all leaf routers.
* In addition to [what happens before the configuration](#network-topology-before-configuration), the Porter replica installed under leaf 2 also establishes a BGP connection with leaf 2 and publishes the worker 2 IP address 192.168.1.2 to leaf 2 as the next hop destined for the service IP address 172.22.0.2.
* Leaf 2 establishes a BGP connection with the spine router and publishes its own IP address 192.168.1.1 to the spine router as the next hop destined for the service IP address 172.22.0.2.
* When an external client machine attempts to access the service, the spine router load balances the service traffic among leaf 1 and leaf 2. Leaf 1 load balances the traffic among the master node and worker 1. Leaf 2 forwards the traffic to worker 2. Therefore, the service traffic is load balanced among all three Kubernetes cluster nodes, and the service bandwidth of all three nodes can be utilized.

## Configuration Procedure

### Prerequisites

You need to prepare a Kubernetes cluster where Porter has been installed. For details, see [Install Porter on Kubernetes (kubectl and Helm)](./install-porter-on-kubernetes.md) and [Install Porter on KubeSphere (Web Console)](./install-porter-on-kubesphere.md).

### Procedure

{{< notice note >}}

The node names, leaf router names, and namespace in the following steps are examples only. You need to use the actual values in your environment.

{{</ notice >}}

1. Log in to the Kubernetes cluster and run the following commands to label the Kubernetes cluster nodes where Porter is to be installed:

   ```bash
   kubectl label --overwrite nodes master1 worker-p002 lb.kubesphere.io/v1alpha1=porter
   ```

   {{< notice note >}}

   Porter works properly if it is installed on only one node under each leaf router. In this example, Porter will be installed on master1 under leaf1 and worker-p002 under leaf2. However, to ensure high availability in a production environment, you are advised to installed Porter on at least two nodes under each leaf router.

   {{</ notice >}}

2. Run the following command to scale the number of porter-manager pods to 0:

   ```bash
   kubectl scale deployment porter-manager --replicas=0 -n porter-system
   ```

3. Run the following command to edit the porter-manager deployment:

   ```bash
   kubectl edit deployment porter-manager -n porter-system
   ```

4. In the porter-manager deployment YAML configuration, add the following fields under `spec.template.spec`:

   ```yaml
   nodeSelector:
     kubernetes.io/os: linux
     lb.kubesphere.io/v1alpha1: porter
   ```
   
5. Run the following command to scale the number of porter-manager pods to the required number (change the number `2` to the actual value):

   ```bash
   kubectl scale deployment porter-manager --replicas=2 -n porter-system
   ```

6. Run the following command to check whether Porter has been installed on the required nodes.

   ```bash
   kubectl get po -n porter-system -o wide
   ```
   
   ![verify-configuration-result](./img/configure-porter-for-multi-router-clusters/verify-configuration-result.jpg)

7. Run the following commands to label the Kubernetes cluster nodes so that the Porter replicas establish BGP connections with the correct BGP routers.

   ```bash
   kubectl label --overwrite nodes master1 porter.kubesphere.io/rack=leaf1
   kubectl label --overwrite nodes worker-p002 porter.kubesphere.io/rack=leaf2
   ```

8. When creating BgpPeer objects, configure the [spec.nodeSelector.matchLabels](./configure-porter-in-bgp-mode.md/#configure-peer-bgp-properties-using-bgppeer) field in the BgpPeer YAML configuration for each leaf router. The following YAML configurations specify that the Porter replica on master1 communicates with leaf1, and the Porter replica on worker-p002 communicates with leaf2. 

   ```yaml
   # BgpPeer YAML for master1 and leaf1
   nodeSelector:
       matchLabels:
         porter.kubesphere.io/rack: leaf1
   ```
   
   ```yaml
   # BgpPeer YAML for worker-p002 and leaf2
   nodeSelector:
       matchLabels:
         porter.kubesphere.io/rack: leaf2
   ```
   
   




