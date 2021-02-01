# Use Porter in Layer 2 Mode

This document demonstrates how to use Porter in Layer 2 mode to expose a service backed by two pods. The Eip, workloads and service described in this document are examples only.

In the Layer 2 mode, you do not need to configure Porter. For details about the network topology of Porter in Layer 2 mode, see [Layer 2 Mode Network Topology](./layer-2-mode-network-topology.md).

## Prerequisites

You need to prepare a Kubernetes cluster where Porter has been installed. For details, see [Install Porter on Kubernetes (kubectl and Helm)](https://github.com/Patrick-LuoYu/porter/blob/configure-porter-for-multi-router-clusters-en/doc/install-porter-on-kubernetes.md) and [Install Porter on KubeSphere (Web Console)](https://github.com/Patrick-LuoYu/porter/blob/configure-porter-for-multi-router-clusters-en/doc/install-porter-on-kubesphere.md).

## Step 1: Enable strictARP for kube-proxy

In Layer 2 mode, you need to enable strictARP for kube-proxy so that all NICs in the Kubernetes cluster stop answering ARP requests from other NICs. Porter handles ARP requests instead.

1. Run the following command to edit the kube-proxy ConfigMap:

   ```bash
   kubectl edit configmap kube-proxy -n kube-system
   ```

2. In the kube-proxy ConfigMap YAML configuration, set `data.config.conf.ipvs.strictARP` to `true`.

   ```yaml
   apiVersion: kubeproxy.config.k8s.io/v1alpha1
   kind: KubeProxyConfiguration
   mode: "ipvs"
   ipvs:
     strictARP: true
   ```

3. Run the following command to restart kube-proxy

   ```bash
   kubectl rollout restart daemonset kube-proxy -n kube-system
   ```

## Step 2: Specify the NIC Used for Porter

If the node where Porter is installed has multiple NICs, you need to specify the NIC used for Porter in Layer 2 mode. You can skip this step if the node has only one NIC.

In the following example, the master1 node where Porter is installed has two NICs (eth0 192.168.0.2 and eth1 192.168.1.2), and eth0 192.168.0.2 will be used for Porter.

Run the following command to annotate master1 to specify the NIC:

```bash
kubectl annotate nodes master1 layer2.porter.kubesphere.io/v1alpha1="192.168.0.2"
```

## Step 3: Create an Eip Object

The Eip object functions as an IP address pool.

1. Run the following command to create a YAML file for the Eip object:

   ```bash
   vi porter-layer2-eip.yaml
   ```

2. Add the following information to the YAML file:

   ```yaml
   apiVersion: network.kubesphere.io/v1alpha2
   kind: Eip
   metadata:
     name: porter-layer2-eip
   spec:
     address: 192.168.0.91-192.168.0.100
     interface: eth0
     protocol: layer2
   ```

   {{< notice note >}}

   * The IP addresses specified in `spec.address` must be on the same network segment as the Kubernetes cluster nodes.

   * For details about fields in the Eip YAML configuration, see [Configure IP Address Pools Using Eip](./configure-ip-address-pools-using-eip.md).

   {{</ notice>}}

3. Run the following command to create the Eip object:

   ```bash
   kubectl apply -f porter-layer2-eip.yaml
   ```

## Step 4: Create a Deployment

The following creates a deployment of two pods using the luksa/kubia image. The pods return their own pod name to external requests. 

1. Run the following command to create a YAML file for the deployment:

   ```bash
   vi porter-layer2.yaml
   ```

2. Add the following information to the YAML file:

   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: porter-layer2
   spec:
     replicas: 2
     selector:
       matchLabels:
         app: porter-layer2
     template:
       metadata:
         labels:
           app: porter-layer2
       spec:
         containers:
           - image: luksa/kubia
             name: kubia
             ports:
               - containerPort: 8080
   ```

3. Run the following command to create the deployment:

   ```bash
   kubectl apply -f porter-layer2.yaml
   ```

## Create a Service

1. Run the following command to create a YAML file for the service:

   ```bash
   vi porter-layer2-svc.yaml
   ```

2. Add the following information to the YAML file:

   ```yaml
   kind: Service
   apiVersion: v1
   metadata:
     name: porter-layer2-svc
     annotations:
       lb.kubesphere.io/v1alpha1: porter
       protocol.porter.kubesphere.io/v1alpha1: layer2
   spec:
     selector:
       app: porter-layer2
     type: LoadBalancer
     ports:
       - name: http
         port: 80
         targetPort: 80
     externalTrafficPolicy: Cluster
   ```

   {{< notice note >}}

   * The `lb.kubesphere.io/v1alpha1: porter` annotation specifies that the service uses Porter.
   * The `protocol.porter.kubesphere.io/v1alpha1: layer2` annotation specifies that the service uses Porter in Layer 2 mode.
   * You must set `spec.type` to `LoadBalancer`.
   * When `spec.externalTrafficPolicy` is set to `Cluster` (default value), Porter randomly selects a node in the Kubernetes cluster to handle service requests. Pods on other nodes can also be reached over kube-proxy.
   * When `spec.externalTrafficPolicy` is set to `Local`, Porter randomly selects a node that contains a pod in the Kubernetes cluster to handle service requests. Only pods on the node can be reached.

   {{</ notice>}}

## Verify layer2

**Note: To verify the layer2 mode, you need to operate on a separate node outside the Kubernetes cluster **.

* View Service LoadBalancer IP
```bash
root@node1:~# kubectl get svc mylbapp-svc-layer2
NAME                   TYPE           CLUSTER-IP    EXTERNAL-IP    PORT(S)        AGE
mylbapp-svc-layer2   LoadBalancer   10.233.44.8   172.22.0.188   80:30564/TCP   10d
```

* View MAC address for Eip

```bash
root@i-7iisycou:~# ping 172.22.0.188
PING 172.22.0.188 (172.22.0.188) 56(84) bytes of data.
64 bytes from 172.22.0.188: icmp_seq=1 ttl=64 time=14.7 ms
64 bytes from 172.22.0.188: icmp_seq=2 ttl=64 time=1.04 ms
^C
--- 172.22.0.188 ping statistics ---
2 packets transmitted, 2 received, 0% packet loss, time 1001ms
rtt min/avg/max/mdev = 1.048/7.911/14.775/6.864 ms
root@i-7iisycou:~# ip neigh
172.22.0.188 dev eth0 lladdr 52:54:22:40:2a:66 DELAY
172.22.0.3 dev eth0 lladdr 52:54:22:40:2a:66 DELAY
```

```bash
root@node1:~# kubectl get nodes -o wide
NAME    STATUS   ROLES           AGE   VERSION   INTERNAL-IP   EXTERNAL-IP   OS-IMAGE             KERNEL-VERSION       CONTAINER-RUNTIME
node1   Ready    master,worker   18d   v1.17.9   172.22.0.3    <none>        Ubuntu 18.04.4 LTS   4.15.0-109-generic   docker://19.3.6
node3   Ready    worker          18d   v1.17.9   172.22.0.9    <none>        Ubuntu 18.04.4 LTS   4.15.0-108-generic   docker://19.3.8
node4   Ready    worker          18d   v1.17.9   172.22.0.10   <none>        Ubuntu 18.04.4 LTS   4.15.0-101-generic   docker://19.3.8
root@node1:~#
```

The above operation reveals that the next hop for 172.22.0.188 is 172.22.0.3, since they both have the same MAC address and point to the same node, node1.

