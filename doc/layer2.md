# Porter Layer2 Usage
Porter now support bgp and layer2, and will support non-bgp switch in future. In
order to distinguish them, file `protocol` was added. 

An example of how to use it will follow.

Create a layer2 EIP
```yaml
kubectl apply -f - <<EOF
apiVersion: network.kubesphere.io/v1alpha1
kind: Eip
metadata:
    name: eip-sample-pool
spec:
    address: 10.11.11.0/24
    protocol: layer2
    disable: false
EOF
```

when we use layer2 EIP in service, annotation "protocol.porter.kubesphere.io/v1alpha1: layer2"
should be provided, unless we specify a fixed IP.
```yaml
kind: Service
apiVersion: v1
metadata:
    name:  mylbapp
    annotations:
        lb.kubesphere.io/v1alpha1: porter
        #eip.porter.kubesphere.io/v1alpha1: 1.1.1.1 
        #protocol.porter.kubesphere.io/v1alpha1: layer2 
spec:
    selector:
        app:  mylbapp
    type:  LoadBalancer 
    ports:
      - name:  http
        port:  8088
        targetPort:  80
    ```
