# Deploy Porter on Bare Metal Kubernetes Cluster

> English | [中文](zh/deploy_baremetal.md)

## Prerequisites
1.  A Kubernetes cluster
1.  Your router needs to support the BGP protocol
1.  Your router needs to support Equal-cost multi-path routing (ECMP) if you want to enable load-balancing on the router. Including the following features:
    - Support multi-path routing
    - Support BGP Additional-Paths
1. If there is a router that does not support the BGP protocol (or is not allowed to enable the BGP protocol), you need to manually write the nexthop route of EIP on this router (or use other routing protocols)

## Install Porter
1. Install Kubernetes Cluster
2. Get Porter's YAML file
     ```bash
    wget https://github.com/kubesphere/porter/releases/download/v0.1.1/porter.yaml
     ```
3. You need to modify a `ConfigMap` named `bgp-cfg` in the YAML according to the [BGP Configuration](bgp_config.md)
4. Install Porter on k8s cluster
     ```bash
     kubectl apply -f porter.yaml
     ```

## Router Configuration
> Different routers have different configurations. Here is the configuration of a Cisco Nexus 9000 Series. For more router configuration, please refer to [Router Configuration](router_config.md).

### [Cisco Nexus 9000 Series](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/92x/unicast/configuration/guide/b-cisco-nexus-9000-series-nx-os-unicast-routing-configuration-guide-92x/b-cisco-nexus-9000-series-nx-os-unicast-routing-configuration-guide-92x_chapter_01010.html)


1. Enter the N9K configuration interface as admin. Modify the following configuration according to the actual situation. 

   ```
    feature bgp

    router bgp 65001
    router-id 10.10.12.1
    address-family ipv4 unicast 
        maximum-paths 8
        additional-paths send
        additional-paths receive
    neighbor 10.10.12.5
        remote-as 65000
        timers 10 30
        address-family ipv4 unicast
        route-map allow in
        route-map allow out
        soft-reconfiguration inbound always
        capability additional-paths receive
    ```

2. After the configuration is complete, check the neighbor status as `Established`.

    ```bash
    myswitvh(config)# show bgp ipv4 unicast neighbors

        BGP neighbor is 10.10.12.5, remote AS 65000, ebgp link, Peer index 3
        BGP version 4, remote router ID 10.10.12.5
        BGP state = Established, up for 00:00:02
        Peer is directly attached, interface Ethernet1/1
        Last read 00:00:01, hold time = 30, keepalive interval is 10 seconds
        Last written 0.996717, keepalive timer expiry due 00:00:09
        Received 5 messages, 0 notifications, 0 bytes in queue
        Sent 13 messages, 0 notifications, 0(0) bytes in queue
        Connections established 1, dropped 0
        Last reset by us 00:01:29, due to session closed
        Last reset by peer never, due to No error

        Neighbor capabilities:
        Dynamic capability: advertised (mp, refresh, gr)
        Dynamic capability (old): advertised
        Route refresh capability (new): advertised received
        Route refresh capability (old): advertised
        4-Byte AS capability: advertised received
        Address family IPv4 Unicast: advertised received
        Graceful Restart capability: advertised
    ```

## Deployment
1.  Add an EIP pool
   
    ```bash
    kubectl apply -f - <<EOF
    apiVersion: network.kubesphere.io/v1alpha1
    kind: Eip
    metadata:
        name: eip-sample-pool
    spec:
        address: 10.11.11.0/24
    EOF
    ```
    Sample: [EIP](https://github.com/kubesphere/porter/blob/master/test/samples/eip.yaml)   

    **Note: EIP address now supports 3 types:**
   
    - IP Address         
        `192.168.0.1`
    - IP Network segment 
        `192.168.0.0/24`
    - IP Range     
        `192.168.0.1-192.168.0.10`

   

2. Deploy a service in kubernetes. The Service must add the following annotations, and the type must also be specified as LoadBalancer:

    ```yaml
    kind: Service
    apiVersion: v1
    metadata:
        name:  mylbapp
        annotations:
            lb.kubesphere.io/v1alpha1: porter
    spec:
        selector:
            app:  mylbapp
        type:  LoadBalancer 
        ports:
        - name:  http
          port:  8088
          targetPort:  80
    ```
    Sample: [Service](https://github.com/kubesphere/porter/blob/master/test/samples/test.yaml)  

    **Note: If you want to assign an IP address to Service, there are two ways:**
    - Add `spec.loadBalancerIP: <ip>` . (recommended)
    - Add `eip.porter.kubesphere.io/v1alpha1: <ip>` to `annotations`.

    


3. On the router we can see that a new network (external IP address) was added with three paths. Each path is linked to one of the nodes:

    ```
    # show bgp all 
 
    10.11.11.11/32, ubest/mbest: 3/0
    *via 10.10.12.2, [20/0], 00:03:38, bgp-65001, external, tag 65000
    *via 10.10.12.3, [20/0], 00:03:38, bgp-65001, external, tag 65000
    *via 10.10.12.4, [20/0], 00:03:38, bgp-65001, external, tag 65000

    ```
4. Use `kubectl get eip` to watch the current usage of EIP