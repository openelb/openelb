# Simulate with Bird

> English | [中文](zh/simulate_with_bird.md)

> This article is an experiment done on [QingCloud](https://www.qingcloud.com/) platform. There are some differences in configuration for different platforms. The advantage of simulation is that you can experience the function of Porter without touching the actual hardware, but it is still different from the real router. The simulated router has a default network card (not used for routing), which will cause the default route when the packet goes back. In addition, when configuring the simulated router, there are many additional parameters, please set these parameters according to this article.

## Prerequisites
1. A k8s cluster
2. Ensure that the host used to simulate the router can be connected to the k8s cluster, including the bgp port and the application port in the cluster.
3. A public IP address


## Create a router

1. Create a host in the k8s network and install Bird on the host. The QingCloud platform only has Bird 1.5 version. This version does not support ECMP. To experience all the functions of Porter, you need to install at least 1.6 version. Execute the following script to install Bird 1.6.
    ```
     $sudo add-apt-repository ppa:cz.nic-labs/bird
     $sudo apt-get update 
     $sudo apt-get install bird
     $sudo systemctl enable bird  
    ```

2. Configure the router's BGP service. Modify `/etc/bird/bird.conf` and add the following parameters:
    ```
    protocol bgp mymaster {   
        description "192.168.1.4";  # Router ID, usually the main IP address
        local as 65001;             # Local AS number, must be different from the AS number of the k8s cluster
        neighbor 192.168.1.5 port 17900 as 65000;  # Master node IP and AS number
        source address 192.168.1.4;                # Router IP  
        import all;   
        export all;
        enable route refresh off;  # Due to the low BGP protocol of bird 1.6, multiple routes advertised by Porter will become a single route, this parameter can be used as a workaround to fix this problem.
        add paths on; # When this parameter is set to on, you can receive multiple routes from the Porter.
    }
     ```
   The above parameters configure a neighbor to the simulated router. The neighbor is the master node of the cluster. **We assume that your Porter controller is deployed on the master node. If you do not want to restrict the porter to be deployed on the master node or the master cannot deploy pods, then you need to add all neighbor nodes to this configuration file according to the above rules.** Modify the `kernel` part of the file, cancel the `export all` comments, and enable the ECMP function:
   ```
   protocol kernel {
        scan time 60;
        import none;
        export all;     # Actually insert routes into the kernel routing table
        merge paths on; # Enable ECMP, this parameter requires at least bird 1.6
   }

   ```

3. Reboot Bird
   ```bash
    $sudo systemctl restart bird 
   ```

4. Configure the Elastic IP. Apply an Elastic IP that **Associate Mode is Internal** on the QingCloud console. Please note that EIP must be an internal associate mode. If it is an external associate mode, EIP cannot be found in VM. Associate this EIP to the VM where the simulated router is located. You only need to complete the associate. There is no need to follow up with the QingCloud documentation.

5. A new network card (usually eth1) will be created when the IP is bound to the QingCloud host. Run `ip a` on the host to check whether the network card is `UP`.  If not, execute `ip link set up eth1`. 

6. Turn on port forwarding and turn off packet filtering on the simulated router 
    ```
    sysctl -w net.ipv4.ip_forward=1   
    sysctl -w net.ipv4.conf.all.rp_filter=0    
    sysctl -w net.ipv4.conf.eth1.rp_filter=0
    sysctl -w net.ipv4.conf.eth0.rp_filter=0
    ```
7. Configure the firewall. Do some port tests on QingCloud console.

8. Configure routing rules. Since the default network card of the simulated router is `eth0`, after the cluster returns the ip packet, it will be sent from `eth0` by default. The user accesses the public network IP from `eth1`, which will cause the transmission to fail. Therefore, the packets sent from the bound IP need to be routed to `eth1`. Access this address on the browser, and use `tcpdump -i eth1` to capture packets on the simulated router, and observe the address of the upper router, such as:

    ```bash
    root@i-7bwamgny:~# tcpdump -i eth1
    tcpdump: verbose output suppressed, use -v or -vv for full protocol decode
    listening on eth1, link-type EN10MB (Ethernet), capture size 262144 bytes
    14:24:07.401555 IP 139.198.254.4.1395 > 139.198.121.228.omniorb: Flags [S], seq 3677905607, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532475097 ecr 0], length 0
    14:24:07.403573 IP 139.198.254.4.1396 > 139.198.121.228.omniorb: Flags [S], seq 2462558694, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532475100 ecr 0], length 0
    14:24:07.654341 IP 139.198.254.4.1397 > 139.198.121.228.omniorb: Flags [S], seq 1471601642, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532475350 ecr 0], length 0
    14:24:10.400770 IP 139.198.254.4.1395 > 139.198.121.228.omniorb: Flags [S], seq 3677905607, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532478097 ecr 0], length 0
    14:24:10.404100 IP 139.198.254.4.1396 > 139.198.121.228.omniorb: Flags [S], seq 2462558694, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532478100 ecr 0], length 0
    14:24:10.658557 IP 139.198.254.4.1397 > 139.198.121.228.omniorb: Flags [S], seq 1471601642, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532478351 ecr 0], length 0
    14:24:16.401591 IP 139.198.254.4.1395 > 139.198.121.228.omniorb: Flags [S], seq 3677905607, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532484098 ecr 0], length 0
    14:24:16.404605 IP 139.198.254.4.1396 > 139.198.121.228.omniorb: Flags [S], seq 2462558694, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532484101 ecr 0], length 0
    14:24:16.656750 IP 139.198.254.4.1397 > 139.198.121.228.omniorb: Flags [S], seq 1471601642, win 64240, options [mss 1394,nop,wscale 8,sackOK,TS val 532484351 ecr 0], length 0

    ```
    In the above output, `139.198.121.228` is the bound IP, and the left side of the `>` is the address of the upper router. Configure the rules for returning packets through routing policies:
    ```bash
    sudo ip rule add from 139.198.254.4/32 lookup 101 # If the packet comes from this IP, then go to the routing table 101
    sudo ip route replace default dev eth1 table 101  # Set the default network card of routing table 101 to eth1
    ```
    The actual physical router does not need to configure the above rules, because the router knows how to configure this IP correctly. **If you need to access and test ECMP from multiple IP addresses, then these IPs also need to be configured with the same steps**
    
9.  After the configuration is completed, you can execute `birdc show protocol` to view the connection information.

> Note: If the way to connect to the host is through the public IP, then after performing the above operation, it is possible that the SSH connection will be disconnected (if and only if the public IP of SSH and the public IP of your test are in QingCloud The network will be NAT to the above 139.198.254.4/32). After disconnection, you can use the VNC to connect the host on the QingCloud website. It is recommended to use VPN connection. The following operations in the k8s cluster will have the same effect.

## Configure plugins
> All the operations are in the master node of the k8s cluster

1. Get Porter's YAML file
    ```
    wget https://github.com/kubesphere/porter/releases/download/v0.1.1/porter.yaml
    ```
2. You need to modify a `ConfigMap` named `bgp-cfg` in the YAML according to the [BGP Configuration](bgp_config.md)
3. Configure public network IP routing rules. Same as the problem with the simulated router. **This step needs to be configured on all k8s nodes, because the actual service may be deployed on any node.**
    ```bash
    sudo ip rule add to 139.198.254.0/24 lookup 101 
    sudo ip route replace default via 192.168.98.5 dev eth0 table 101 # 192.168.98.5 is the router IP
    ```
    The above `192.168.98.5` is the address of the simulated router. A loop rule has been configured on the simulated router, so this packet will not be dropped. The actual k8s cluster does not need to be configured, because the default gateway of the k8s cluster is this router.

4. Install Porter on k8s cluster: `kubectl apply -f porter.yaml`
5. Add and EIP
   ```bash
   kubectl apply -f - <<EOF
    apiVersion: network.kubesphere.io/v1alpha1
    kind: Eip
    metadata:
        name: eip-sample
    spec:
        address: 139.198.121.228
        protocol: bgp
        disable: false
    EOF 
   ```
6. Deploy a service in kubernetes. The Service must add the following annotations, and the type must also be specified as LoadBalancer:

    ```yaml
    kind: Service
    apiVersion: v1
    metadata:
    name:  mylbapp
    annotations:
        lb.kubesphere.io/v1alpha1: porter
        #protocol.porter.kubesphere.io/v1alpha1: bgp 
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

7. Check the Porter logs and EIP events, if there is no problem, you can access the service according to the EIP.
   ```bash
   kubectl logs -n porter-system controller-manager-0 -c manager # Check the logs of Porter
   kubectl describe eip eip-sample # Check the events
   ```
8. Check if there are two equal-cost routes on the simulated router:
   ```bash
   root@i-7bwamgny:~# ip route
   default via 192.168.98.1 dev eth0
   139.198.121.228  proto bird
        nexthop via 192.168.98.2  dev eth0 weight 1
        nexthop via 192.168.98.4  dev eth0 weight 1
   ```

## Test the load balancing of ECMP
> Note: The kernel version of the host where the simulated router is located must be higher than 3.6. The default kernel version of QingCloud platform is 4.4. The ECMP Hash algorithm used is `L3`. ECMP will only adjust the access route based on the source IP. The kernel versions above 4.12 support `L4`. You can run  `sysctl net.ipv4.fib_multipath_hash_policy 1` to change the load balancing hash algorithm, then use `curl` to access this eip, so that you can achieve the effect of load balancing.

The actual router only needs to enable the ECMP function to achieve load balancing. In order to test the effectiveness of load balancing, you need to access this EIP from different source IPs and observe whether there is traffic on each node.

1. Observe the node where the Pod is located
```bash
kubectl get pod -o wide
```
2. Observe whether each node has set routing rules
```bash
root@master-k8s:~# ip rule
0:      from all lookup local
32763:  from all to 139.198.121.228 lookup 101
```
3. Run `tcpdump -i eth0 port $port` on these nodes, where `$port` is the port exposed by the service. In the above example, it is 8088.
4. Access this EIP from different IPs and observe whether there is traffic on these nodes. If there is any, then there is no problem with load balancing.

