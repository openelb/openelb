#EIP Configuration

Porter only supports ipv4 address now. Let's take this as an example

```yaml
apiVersion: network.kubesphere.io/v1alpha1
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    address: 192.168.0.0/24
    protocol: layer2
```

1. The protocol should be layer 2 or BGP.
2. The address supports three kinds of syntax.
```yaml
- ip        e.g.  192.168.0.1
- ip/net    e.g.  192.168.0.0/24
- ip1-ip2   e.g.  192.168.0.1-192.168.0.10
```