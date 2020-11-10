# Eip Configuration

> English | [中文](zh/eip_config.md)


Eip is used to configure IP address segments, which Porter will assign to the LoadBalancer Service, and then publish routes via `BGP/ARP/NDP` protocols. 

**Note: Porter currently only supports IPv4 addresses, support for IPv6 will be completed soon.**

Eip's example below shows all available configuration fields and a description of the status fields.

```yaml
apiVersion: network.kubesphere.io/v1alpha2
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    address: 192.168.0.0/24
    protocol: layer2
    interface: eth0
    disable: false
status:
    occupied: false
    usage: 1
    poolSize: 256
    used: 
      "192.168.0.1": "default/test-svc"
    firstIP: 192.168.0.0
    lastIP: 192.168.0.255
    ready: true
    v4: true
```

## spec field explanation

* address

`address` is used to describe a range of IP addresses, which can have the following three formats

```yaml
- ip        e.g.  192.168.0.1
- ip/net    e.g.  192.168.0.0/24
- ip1-ip2   e.g.  192.168.0.1-192.168.0.10
```

**Note: The IP address segment must not overlap with other created Eip, otherwise the resource creation error will occur.**

* protocol

`protocol` is used to describe what protocol is used to publish routes, and the valid values are `layer2` and `bgp`. When the value is null, the mode protocol is `bgp`.

* interface

`interface` makes sense when `protocol` is `layer2` and is used to indicate which network card Porter is listening for ARP/NDP requests on.

When the NIC names in each node of a Kubernetes cluster are different, you can specify the NIC by using the syntax `interface: can_reach:192.168.1.1`. In the above example, Porter gets the first NIC in the route by finding the route to 192.168.1.1.

* disable

With `true`, Porter will not be assigned an address from this Eip when a new LoadBalancer Service is created, but it will not affect a Service already created.

## status field explanation

* occupied

This field is used to indicate whether or not an address in Eip has been allocated and used up.

* usage 和 used

`usage` is used to indicate how many addresses have been allocated in Eip; `used` is used to indicate which address is being used by which Service, key is the IP address, value is the Service's `Namespace/Name`.

* poolSize

This field is used to indicate the total number of addresses in Eip.

* firstIP

This field is used to represent the first IP address in Eip.

* lastIP

This field is used to represent the last IP address in Eip.

* v4

This field is used to represent the address protocol family of Eip.

* ready

This field is used to indicate whether the BGP/ARP/NDP related program associated with Eip has been initialized or not.