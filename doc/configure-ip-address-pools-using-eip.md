# Configure IP Address Pools Using Eip

This document describes how to configure an Eip object, which functions as an IP address pool for Porter both in BGP mode and in Layer 2 mode.

Porter assigns IP addresses in Eip objects to LoadBalancer services in the Kubernetes cluster. After that, Porter publishes routes destined for the service IP addresses over BGP (in BGP mode), ARP (in Layer 2 mode for IPv4), or NDP (in Layer 2 mode for IPv6). 

{{< notice note>}}

Currently, Porter supports only IPv4 and will soon support IPv6.

{{</ notice>}}

## Configure an Eip Object for Porter

You can create an Eip object to provide an IP address pool for Porter. The following is an example of the Eip YAML configuration:

```yaml
apiVersion: network.kubesphere.io/v1alpha2
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    address: 192.168.0.91-192.168.0.100
    protocol: layer2
    interface: eth0
    disable: false
status:
    occupied: false
    usage: 1
    poolSize: 10
    used: 
      "192.168.0.91": "default/test-svc"
    firstIP: 192.168.0.91
    lastIP: 192.168.0.100
    ready: true
    v4: true
```

The fields are described as follows:

`metadata`:

* `name`: Name of the Eip object.

`spec`:

* `address`: One or more IP addresses, which will be used by Porter. The value format can be:
  
  * `IP address`, for example, `192.168.0.100`.
  * `IP address/Subnet mask`, for example, `192.168.0.0/24`.
* `IP address 1-IP address 2`, for example, `192.168.0.91-192.168.0.100`.
  

  {{< notice note>}}

  IP segments in different Eip objects cannot overlap. Otherwise, a resource creation error will occur.

  {{</ notice>}}


* `protocol`: Specifies which mode of Porter the Eip object is used for. The value can be either `layer2` or `bgp`. If this field is not specified, the default value `bgp` is used.

* `interface`: NIC on which Porter listens ARP or NDP requests. This field is valid only when `protocol` is set to `layer2`.

  {{< notice tip >}}

  If the NIC names of the Kubernetes cluster nodes are different, you can set the value to `can_reach:IP address` (for example, `can_reach:192.168.0.5`) so that Porter automatically obtains the name of the NIC that can reach the IP address. In this case, you must ensure that the IP address is not used by Kubernetes cluster nodes but can be reached by the cluster nodes.

  {{</ notice >}}

* `disable`: Specifies whether the Eip object is disabled. The value can be:
  
  * `false`: Porter can assign IP addresses in the Eip object to new LoadBalancer services.
  * `true`: Porter stops assigning IP addresses in the Eip object to new LoadBalancer services. Existing services will not be affected.

`status`: Fields under `status` specify the status of the Eip object and are automatically configured. When creating an Eip object, you do not need to configure these fields.

* `occupied`: Specifies whether IP addresses in the Eip object has been used up.

* `usage`: Specifies how many IP addresses in the Eip object have been assigned to services.
* `used`: Specifies the used IP addresses and the services that use the IP addresses. The services are displayed in the `Namespace/Service name` format (for example, `default/test-svc`).

* `poolSize`: Total number of IP addresses in the Eip object.

* `firstIP`: First IP address in the Eip object.

* `lastIP`: Last IP address in the Eip object.

* `v4`: Specifies whether the address family is IPv4. Currently, Porter supports only IPv4 and the value can only be `true`.

* `ready`: Specifies whether the Eip-associated program used for BGP/ARP/NDP routes publishing has been initialized. The program is integrated in Porter.
