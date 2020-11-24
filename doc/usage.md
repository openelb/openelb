# Usage

> English | [中文](zh/usage.md)


After Porter is installed and configured, you can use Porter exposure by creating a LoadBalancer Service.

## Configuring LoadBalancer Service

The LoadBalancer Service must add annotations `lb.kubesphere.io/v1alpha1: porter` and the type must be specified as `LoadBalancer`.

```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
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

**Porter does not handle other non-qualified Services. **

### Specify Protocol

You can specify the protocol by which Porter assigns addresses by specifying annotation "protocol.porter.kubesphere.io/v1alpha1".

The following example specifies annotation "protocol.porter.kubesphere.io/v1alpha1: bgp", which means that Porter will allocate the address from the BGP protocol's Eip. It is also the default value for Porter and can be omitted.

```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: bgp
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  ports:
    - name:  http
      port:  8088
      targetPort:  80
```

The following example specifies annotation "protocol.porter.kubesphere.io/v1alpha1: layer2", which means that Porter will allocate the address from the layer2 protocol's Eip.

```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: layer2
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  ports:
    - name:  http
      port:  8088
      targetPort:  80
```

### Specify Eip

Suppose we have the following example of Eip

```yaml
apiVersion: network.kubesphere.io/v1alpha2
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    address: 192.168.0.0/24
```

Porter iterates through all `enable` Eip's when processing the LoadBalancer Service, **allocating addresses from the first Eip that matches the protocol and has a free address**, which means that the allocated address will be unpredictable. Service specifies the parameter `spec.loadBalancerIP` to fix the address.

```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: bgp
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  loadBalancerIP: 192.168.0.100
  ports:
    - name:  http
      port:  8088
      targetPort:  80
```

If you don't need a fixed address, but just need to allocate addresses from a specific address pool, then you can specify annotation "eip.porter.kubesphere.io/v1alpha2", which takes the value of Eip's Name.

```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: bgp
    eip.porter.kubesphere.io/v1alpha2: eip-sample-pool
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  ports:
    - name:  http
      port:  8088
      targetPort:  80
```

### Configure spec.externalTrafficPolicy

By default, the LoadBalancer Service's `externalTrafficPolicy` value is `Cluster`, which means that all nodes in a Kubernetes cluster can be used as Nexthop to forward traffic, the difference is simply that.
* In BGP mode, Porter publishes equivalent routes containing all Nodes as Nexthop.
* Under layer2, Porter randomly selects a Node as a Nexthop to forward traffic.

Kubernetes also provides an additional value of `Local` for `externalTrafficPolicy`, which means that only Nodes contained in Endpoints in a Kubernetes cluster can be used as Nexthop to forward traffic.

### Share Eip

Generally, the number of exposed public IP addresses is limited, and when there are many services to be exposed, the Eip address will not be enough, **Porter currently supports BGP mode Eip sharing**. To use it, you just need to specify the Eip you want to share as described in the **Specify Eip** section above.

**Note: The two Service ports ports cannot be duplicated, and need externalTrafficPolicy not to be Local**.

The following is an existing Service and specifies `loadBalancerIP: 192.168.0.100`.

```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: bgp
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  loadBalancerIP: 192.168.0.100
  ports:
    - name:  http
      port:  8088
      targetPort:  80
```

Here, if you want to multiplex the 192.168.0.100 IP, you can just create a Service and specify the IP at the same time.

```yaml
kind: Service
apiVersion: v1
metadata:
  name:  mylbapp-svc2
  annotations:
    lb.kubesphere.io/v1alpha1: porter
    protocol.porter.kubesphere.io/v1alpha1: bgp
spec:
  selector:
    app:  mylbapp
  type:  LoadBalancer
  loadBalancerIP: 192.168.0.100
  ports:
    - name:  http
      port:  8089
      targetPort:  80
```